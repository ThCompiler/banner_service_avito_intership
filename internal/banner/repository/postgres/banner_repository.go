package postgres

import (
	"bannersrv/internal/banner"
	"bannersrv/internal/banner/entity"
	"bannersrv/internal/banner/repository"
	"bannersrv/internal/pkg/pg"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/logger"
	"bannersrv/pkg/slices"
	"database/sql"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

const (
	createQuery = `
		INSERT INTO banner (is_active)
		VALUES ($1)
		RETURNING id
	`

	addFeaturesAndTagsQuery = `
		INSERT INTO features_tags_banner (banner_id, feature_id, tag_id)
		SELECT $1, $2, tag
		FROM unnest($3::bigint[]) as tag		
	`

	addContentQuery = `
		INSERT INTO version_banner (banner_id, content) VALUES ($1, $2)
	`

	deleteQuery = `
		DELETE FROM banner WHERE id = $1 RETURNING id
	`

	updateActiveQuery = `
		UPDATE banner SET is_active = $2
			WHERE id = $1 and not deleted
			RETURNING id
	`

	deleteFeaturesTagsQuery = `
		WITH deleted_features AS (
			DELETE FROM features_tags_banner WHERE banner_id = $1 RETURNING feature_id
		)
		SELECT DISTINCT feature_id FROM deleted_features LIMIT 1
	`

	updateFeaturesQuery = `
		UPDATE features_tags_banner SET feature_id = $2 WHERE banner_id = $1
	`

	getQuery = `
		SELECT vb.content FROM banner
		   INNER JOIN features_tags_banner on (features_tags_banner.banner_id = banner.id)
		   LEFT JOIN version_banner as vb on (vb.banner_id = banner.id)
		WHERE is_active and not deleted and vb.version = COALESCE($3::bigint, banner.last_version) 
		  		and feature_id = $1 and tag_id = $2 LIMIT 1
	`

	filterQuery = `
		SELECT banner.id, array_agg(ftb.tag_id)::bigint[] as tag_ids, ftb.feature_id, banner.is_active,
       		banner.created_at, banner.updated_at FROM banner
		INNER JOIN features_tags_banner as ftb ON (ftb.banner_id = banner.id)
		WHERE not deleted and banner.id in (
			(
				SELECT DISTINCT banner_id FROM features_tags_banner
				WHERE (CASE WHEN $1::bigint IS NOT NULL THEN feature_id = $1 ELSE true END)
      			and (CASE WHEN $2::bigint IS NOT NULL THEN tag_id = $2 ELSE true END) 
			)
		)
		GROUP BY banner.id, ftb.feature_id
		LIMIT $3 OFFSET $4
	`

	getVersionQuery = `
		SELECT banner_id, content, version, created_at FROM version_banner WHERE banner_id in (?)
	`
	//	WITH filter_banner AS (
	// SELECT DISTINCT banner_id FROM features_tags_banner
	// WHERE (CASE WHEN $1::bigint IS NOT NULL THEN feature_id = $1 ELSE true END)
	// and (CASE WHEN $2::bigint IS NOT NULL THEN tag_id = $2 ELSE true END)
	// )
	delayedDeletionQuery = `
		UPDATE banner SET deleted = true FROM features_tags_banner 
			 WHERE banner.id = features_tags_banner.banner_id and
				   (CASE WHEN $1::bigint IS NOT NULL THEN feature_id = $1 ELSE true END)
					and (CASE WHEN $2::bigint IS NOT NULL THEN tag_id = $2 ELSE true END) 
	`

	cronDeleteQuery = `
		DELETE FROM banner WHERE deleted
	`
)

const (
	durationToCronDeleteBanner = 5 * time.Hour
)

type BannerRepository struct {
	db             *sqlx.DB
	tasksScheduler gocron.Scheduler
	l              logger.Interface
}

func NewBannerRepository(db *sqlx.DB, tasksScheduler gocron.Scheduler, l logger.Interface) (*BannerRepository, error) {
	br := &BannerRepository{
		db:             db,
		tasksScheduler: tasksScheduler,
		l:              l,
	}

	if _, err := tasksScheduler.NewJob(
		gocron.DurationJob(durationToCronDeleteBanner),
		gocron.NewTask(
			func(rep banner.Repository, l logger.Interface) {
				if err := rep.CleanDeletedBanner(); err != nil {
					l.Error(errors.Wrap(err, "in cron job of cleaning deleted banner"))
				}
				l.Info("deleted banner was cleaned by cron job")
			},
			br,
			l,
		),
	); err != nil {
		return nil, err
	}

	return br, nil
}

func (*BannerRepository) addContent(tx *sqlx.Tx, id types.ID, content types.Content) error {
	if _, err := tx.Exec(addContentQuery, id, content); err != nil {
		return errors.Wrap(err, "can't add content to banner")
	}

	return nil
}

func (br *BannerRepository) CreateBanner(featureID types.ID, tagIDs []types.ID,
	content types.Content, isActive bool,
) (types.ID, error) {
	var createdID types.ID

	if err := pg.WithTransaction(br.db,
		func(tx *sqlx.Tx) error {
			if err := tx.QueryRowx(createQuery, isActive).
				Scan(
					&createdID,
				); err != nil {
				return errors.Wrap(err, "can't create banner")
			}

			if err := br.addContent(tx, createdID, content); err != nil {
				return err
			}

			if _, err := tx.Exec(addFeaturesAndTagsQuery, createdID, featureID, pq.Array(tagIDs)); err != nil {
				return errors.Wrapf(checkPgConflictError(err),
					"can't add feature id %d and tag ids %v to banner", featureID, tagIDs)
			}

			return nil
		},
	); err != nil {
		return 0, errors.Wrap(err, "when creating banner")
	}

	return createdID, nil
}

func (br *BannerRepository) DeleteBanner(id types.ID) (types.ID, error) {
	var deletedID types.ID
	if err := br.db.QueryRowx(deleteQuery, id).
		Scan(
			&deletedID,
		); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return deletedID, errors.Wrapf(repository.ErrorBannerNotFound, "with id %d", id)
		}

		return deletedID, errors.Wrapf(err, "can't delete banner with id %d", id)
	}

	return deletedID, nil
}

func (*BannerRepository) updateBannerInfo(tx *sqlx.Tx, bnr *entity.BannerUpdate) error {
	switch {
	// Если у нас изменился только айди фичи, её можно обновить по id баннера
	case bnr.TagIDs.IsNull && !bnr.FeatureID.IsNull:
		if _, err := tx.Exec(updateFeaturesQuery, bnr.ID, bnr.FeatureID.Value); err != nil {
			return errors.Wrapf(checkPgConflictError(err),
				"can't update feature id %d to banner", bnr.FeatureID.Value)
		}
	// Если у нас изменился список тэгов, то нужно сначала удалить все записи с тэгами, а потом их снова создать
	case !bnr.TagIDs.IsNull:
		var featureID types.ID
		if err := tx.QueryRowx(deleteFeaturesTagsQuery, bnr.ID).Scan(&featureID); err != nil {
			return errors.Wrap(err, "can't delete feature id and tag ids of banner")
		}

		if !bnr.FeatureID.IsNull {
			featureID = bnr.FeatureID.Value
		}

		if _, err := tx.Exec(addFeaturesAndTagsQuery, bnr.ID, featureID, pq.Array(bnr.TagIDs.Value)); err != nil {
			return errors.Wrapf(checkPgConflictError(err),
				"can't add feature id %d and tag ids %v to banner", featureID, bnr.TagIDs.Value)
		}
	}

	return nil
}

func (br *BannerRepository) UpdateBanner(bnr *entity.BannerUpdate) (types.ID, error) {
	var updatedID types.ID

	if err := pg.WithTransaction(br.db,
		func(tx *sqlx.Tx) error {
			if !bnr.IsActive.IsNull {
				if err := tx.QueryRowx(updateActiveQuery, bnr.ID, bnr.IsActive.Value).Scan(&updatedID); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return repository.ErrorBannerNotFound
					}

					return errors.Wrapf(err, "can't update banner")
				}
			}

			if !bnr.Content.IsNull {
				if err := br.addContent(tx, bnr.ID, bnr.Content.Value); err != nil {
					return err
				}
			}

			return br.updateBannerInfo(tx, bnr)
		},
	); err != nil {
		return 0, errors.Wrapf(err, "when updating banner with id %d", bnr.ID)
	}

	return updatedID, nil
}

func (*BannerRepository) filterBanners(tx *sqlx.Tx, bnr *entity.BannerInfo,
	offset, limit uint64,
) ([]entity.Banner, error) {
	rows, err := tx.Queryx(filterQuery,
		&sql.NullInt64{
			Valid: !bnr.FeatureID.IsNull,
			Int64: int64(bnr.FeatureID.Value),
		}, &sql.NullInt64{
			Valid: !bnr.TagID.IsNull,
			Int64: int64(bnr.TagID.Value),
		},
		limit, offset)
	defer func() {
		_ = rows.Close()
	}()

	if err != nil {
		return nil, errors.Wrap(err, "can't execute filter banner query")
	}

	banners := make([]entity.Banner, 0)

	for rows.Next() {
		var filteredBanner entity.Banner

		tagIDs := make([]int64, 0)

		err := rows.Scan(
			&filteredBanner.ID,
			pq.Array(&tagIDs),
			&filteredBanner.FeatureID,
			&filteredBanner.IsActive,
			&filteredBanner.CreatedAt,
			&filteredBanner.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "can't scan filter banner query result")
		}

		filteredBanner.TagIDs = slices.Map(tagIDs, func(id *int64) types.ID { return types.ID(*id) })
		filteredBanner.Versions = make([]entity.Content, 0)

		banners = append(banners, filteredBanner)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "can't end scan filter banner query result")
	}

	return banners, nil
}

func (*BannerRepository) selectContentForBanners(tx *sqlx.Tx, banners []entity.Banner) ([]entity.Banner, error) {
	bannerIDs := make([]types.ID, len(banners))
	bannerIndexes := make(map[types.ID]int64)

	for index, bnr := range banners {
		bannerIDs[index] = bnr.ID
		bannerIndexes[bnr.ID] = int64(index)
	}

	query, args, err := sqlx.In(getVersionQuery, bannerIDs)
	if err != nil {
		return nil, errors.Wrap(err, "can't prepare query to get contents for banner")
	}

	query = tx.Rebind(query)

	rows, err := tx.Queryx(query, args...)
	defer func() {
		_ = rows.Close()
	}()

	if err != nil {
		return nil, errors.Wrap(err, "can't execute get contents for banner query")
	}

	for rows.Next() {
		var bannerContent entity.Content

		var bannerID types.ID

		err := rows.Scan(
			&bannerID,
			&bannerContent.Content,
			&bannerContent.Version,
			&bannerContent.CreatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "can't scan get contents for banner query result")
		}

		banners[bannerIndexes[bannerID]].Versions = append(banners[bannerIndexes[bannerID]].Versions, bannerContent)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "can't end scan get contents for banner query result")
	}

	return banners, nil
}

func (br *BannerRepository) GetBanners(bnr *entity.BannerInfo,
	offset, limit uint64,
) ([]entity.Banner, error) {
	var banners []entity.Banner

	if err := pg.WithTransaction(br.db,
		func(tx *sqlx.Tx) error {
			var err error

			banners, err = br.filterBanners(tx, bnr, offset, limit)
			if err != nil {
				return err
			}

			if len(banners) == 0 {
				return nil
			}

			banners, err = br.selectContentForBanners(tx, banners)
			if err != nil {
				return err
			}

			return nil
		},
	); err != nil {
		return nil, errors.Wrapf(err,
			"when selecting banners for admin with feature id %d, tag id %d, limit %d and offset %d",
			bnr.FeatureID.Value, bnr.TagID.Value, limit, offset)
	}

	return banners, nil
}

func (br *BannerRepository) GetBanner(featureID, tagID types.ID,
	version types.NullableObject[uint32],
) (types.Content, error) {
	var content types.Content
	if err := br.db.QueryRowx(getQuery, featureID, tagID,
		&sql.NullInt64{
			Valid: !version.IsNull,
			Int64: int64(version.Value),
		}).
		Scan(
			&content,
		); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return content, errors.Wrapf(repository.ErrorBannerNotFound,
				"with feature id %d and tag id %d and version %v", featureID, tagID, version)
		}

		return content, errors.Wrapf(err,
			"can't get banner with feature id %d and tag id %d and version %v", featureID, tagID, version)
	}

	return content, nil
}

func (br *BannerRepository) runCronJob() error {
	// Вызываем отложенное удаление баннера
	_, err := br.tasksScheduler.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
		gocron.NewTask(
			func(rep banner.Repository, l logger.Interface) {
				if err := rep.CleanDeletedBanner(); err != nil {
					l.Error(errors.Wrap(err, "in immediately cron job of cleaning deleted banner"))
				}

				l.Info("deleted banner was cleaned by immediately cron job")
			},
			br,
			br.l,
		),
	)
	if err != nil {
		// т.к. не получилось создать задачу удаление, и мы возвращаем ошибку пользователя,
		// надо отменить удаление баннера
		return errors.Wrap(err, "can't start task to delete banner")
	}

	return nil
}

func (br *BannerRepository) DeleteFilteredBanner(bnr *entity.BannerInfo) error {
	if err := pg.WithTransaction(br.db,
		func(tx *sqlx.Tx) error {
			res, err := tx.Exec(delayedDeletionQuery,
				&sql.NullInt64{
					Valid: !bnr.FeatureID.IsNull,
					Int64: int64(bnr.FeatureID.Value),
				}, &sql.NullInt64{
					Valid: !bnr.TagID.IsNull,
					Int64: int64(bnr.TagID.Value),
				})
			if err != nil {
				return errors.Wrap(err, "can't delete banner")
			}

			effected, err := res.RowsAffected()
			if err != nil {
				return errors.Wrap(err, "can't get result of deleting banner")
			}

			if effected == 0 {
				return repository.ErrorBannerNotFound
			}

			return br.runCronJob()
		},
	); err != nil {
		return errors.Wrapf(err,
			"when deleting banner with feature id %d or tag id %d", bnr.FeatureID.Value, bnr.TagID.Value)
	}

	return nil
}

func (br *BannerRepository) CleanDeletedBanner() error {
	_, err := br.db.Exec(cronDeleteQuery)
	if err != nil {
		return errors.Wrap(err, "can't delete deleted banner")
	}

	return nil
}

const (
	uniqueConflictCode   = "23505"
	uniqueConstraintName = "banner_identifier"
)

func checkPgConflictError(err error) error {
	var e *pgconn.PgError

	if !errors.As(err, &e) {
		return err
	}

	if e.Code == uniqueConflictCode && e.ConstraintName == uniqueConstraintName {
		return repository.ErrorBannerConflictExists
	}

	return err
}
