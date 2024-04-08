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
	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	"time"
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
	//SELECT DISTINCT banner_id FROM features_tags_banner
	//WHERE (CASE WHEN $1::bigint IS NOT NULL THEN feature_id = $1 ELSE true END)
	//and (CASE WHEN $2::bigint IS NOT NULL THEN tag_id = $2 ELSE true END)
	//)
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
				if err := br.CleanDeletedBanner(); err != nil {
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

func (br *BannerRepository) addContent(tx *sqlx.Tx, id types.Id, content types.Content) error {
	if _, err := tx.Exec(addContentQuery, id, content); err != nil {
		return errors.Wrap(err, "can't add content to banner")
	}

	return nil
}

func (br *BannerRepository) CreateBanner(featureId types.Id, tagIds []types.Id,
	content types.Content, isActive bool) (types.Id, error) {
	var createdId types.Id
	if err := pg.WithTransaction(br.db,
		func(tx *sqlx.Tx) error {
			if err := tx.QueryRowx(createQuery, isActive).
				Scan(
					&createdId,
				); err != nil {
				return errors.Wrap(err, "can't create banner")
			}

			if err := br.addContent(tx, createdId, content); err != nil {
				return err
			}

			if _, err := tx.Exec(addFeaturesAndTagsQuery, createdId, featureId, pq.Array(tagIds)); err != nil {
				return errors.Wrapf(checkPgConflictError(err),
					"can't add feature id %d and tag ids %v to banner", featureId, tagIds)
			}

			return nil
		},
	); err != nil {
		return 0, errors.Wrap(err, "when creating banner")
	}

	return createdId, nil
}

func (br *BannerRepository) DeleteBanner(id types.Id) (types.Id, error) {
	var deletedId types.Id
	if err := br.db.QueryRowx(deleteQuery, id).
		Scan(
			&deletedId,
		); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return deletedId, errors.Wrapf(repository.ErrorBannerNotFound, "with id %d", id)
		}
		return deletedId, errors.Wrapf(err, "can't delete banner with id %d", id)
	}

	return deletedId, nil
}

func (br *BannerRepository) updateBannerInfo(tx *sqlx.Tx, bnr *entity.BannerUpdate) error {
	switch {
	// Если у нас изменился только айди фичи, её можно обновить по id баннера
	case bnr.TagIds.IsNull && !bnr.FeatureId.IsNull:
		if _, err := tx.Exec(updateFeaturesQuery, bnr.Id, bnr.FeatureId.Value); err != nil {
			return errors.Wrapf(checkPgConflictError(err),
				"can't update feature id %d to banner", bnr.FeatureId.Value)
		}
	// Если у нас изменился список тэгов, то нужно сначала удалить все записи с тэгами, а потом их снова создать
	case !bnr.TagIds.IsNull:
		var featureId types.Id
		if err := tx.QueryRowx(deleteFeaturesTagsQuery, bnr.Id).Scan(&featureId); err != nil {
			return errors.Wrap(err, "can't delete feature id and tag ids of banner")
		}

		if !bnr.FeatureId.IsNull {
			featureId = bnr.FeatureId.Value
		}

		if _, err := tx.Exec(addFeaturesAndTagsQuery, bnr.Id, featureId, pq.Array(bnr.TagIds.Value)); err != nil {
			return errors.Wrapf(checkPgConflictError(err),
				"can't add feature id %d and tag ids %v to banner", featureId, bnr.TagIds.Value)
		}
	}

	return nil
}

func (br *BannerRepository) UpdateBanner(bnr *entity.BannerUpdate) (types.Id, error) {
	var updatedId types.Id
	if err := pg.WithTransaction(br.db,
		func(tx *sqlx.Tx) error {
			if !bnr.IsActive.IsNull {
				if err := tx.QueryRowx(updateActiveQuery, bnr.Id, bnr.IsActive.Value).Scan(&updatedId); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return repository.ErrorBannerNotFound
					}
					return errors.Wrapf(err, "can't update banner")
				}
			}

			if !bnr.Content.IsNull {
				if err := br.addContent(tx, bnr.Id, bnr.Content.Value); err != nil {
					return err
				}
			}

			if err := br.updateBannerInfo(tx, bnr); err != nil {
				return err
			}

			return nil
		},
	); err != nil {
		return 0, errors.Wrapf(err, "when updating banner with id %d", bnr.Id)
	}

	return updatedId, nil
}

func (br *BannerRepository) filterBanners(tx *sqlx.Tx, bnr *entity.BannerInfo,
	offset uint64, limit uint64) ([]entity.Banner, error) {
	rows, err := tx.Queryx(filterQuery,
		&sql.NullInt64{
			Valid: !bnr.FeatureId.IsNull,
			Int64: int64(bnr.FeatureId.Value),
		}, &sql.NullInt64{
			Valid: !bnr.TagId.IsNull,
			Int64: int64(bnr.TagId.Value),
		},
		limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "can't execute filter banner query")
	}

	banners := make([]entity.Banner, 0)

	for rows.Next() {
		var filteredBanner entity.Banner

		tagIds := make([]int64, 0)

		err := rows.Scan(
			&filteredBanner.Id,
			pq.Array(&tagIds),
			&filteredBanner.FeatureId,
			&filteredBanner.IsActive,
			&filteredBanner.CreatedAt,
			&filteredBanner.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "can't scan filter banner query result")
		}

		filteredBanner.TagIds = slices.Map(tagIds, func(id *int64) types.Id { return types.Id(*id) })
		filteredBanner.Versions = make([]entity.Content, 0)

		banners = append(banners, filteredBanner)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "can't end scan filter banner query result")
	}

	return banners, nil
}

func (br *BannerRepository) selectContentForBanners(tx *sqlx.Tx, banners []entity.Banner) ([]entity.Banner, error) {
	bannerIds := make([]types.Id, len(banners))
	bannerIndexes := make(map[types.Id]int64)
	for index, bnr := range banners {
		bannerIds[index] = bnr.Id
		bannerIndexes[bnr.Id] = int64(index)
	}

	query, args, err := sqlx.In(getVersionQuery, bannerIds)
	if err != nil {
		_ = tx.Rollback()
		return nil, errors.Wrap(err, "can't prepare query to get contents for banner")
	}

	query = tx.Rebind(query)

	rows, err := tx.Queryx(query, args...)
	if err != nil {
		_ = tx.Rollback()
		return nil, errors.Wrap(err, "can't execute get contents for banner query")
	}

	for rows.Next() {
		var bannerContent entity.Content
		var bannerId types.Id

		err := rows.Scan(
			&bannerId,
			&bannerContent.Content,
			&bannerContent.Version,
			&bannerContent.CreatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "can't scan get contents for banner query result")
		}

		banners[bannerIndexes[bannerId]].Versions = append(banners[bannerIndexes[bannerId]].Versions, bannerContent)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "can't end scan get contents for banner query result")
	}

	return banners, nil
}

func (br *BannerRepository) GetBanners(bnr *entity.BannerInfo,
	offset uint64, limit uint64) ([]entity.Banner, error) {

	var banners []entity.Banner
	var err error
	if err := pg.WithTransaction(br.db,
		func(tx *sqlx.Tx) error {
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
			bnr.FeatureId.Value, bnr.TagId.Value, limit, offset)
	}

	return banners, nil
}

func (br *BannerRepository) GetBanner(featureId types.Id, tagId types.Id,
	version types.NullableObject[uint32]) (types.Content, error) {
	var content types.Content
	if err := br.db.QueryRowx(getQuery, featureId, tagId,
		&sql.NullInt64{
			Valid: !version.IsNull,
			Int64: int64(version.Value),
		}).
		Scan(
			&content,
		); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return content, errors.Wrapf(repository.ErrorBannerNotFound,
				"with feature id %d and tag id %d and version %v", featureId, tagId, version)
		}
		return content, errors.Wrapf(err,
			"can't get banner with feature id %d and tag id %d and version %v", featureId, tagId, version)
	}

	return content, nil
}

func (br *BannerRepository) runCronJob() error {
	// Вызываем отложенное удаление баннера
	_, err := br.tasksScheduler.NewJob(
		gocron.OneTimeJob(gocron.OneTimeJobStartImmediately()),
		gocron.NewTask(
			func(rep banner.Repository, l logger.Interface) {
				if err := br.CleanDeletedBanner(); err != nil {
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
					Valid: !bnr.FeatureId.IsNull,
					Int64: int64(bnr.FeatureId.Value),
				}, &sql.NullInt64{
					Valid: !bnr.TagId.IsNull,
					Int64: int64(bnr.TagId.Value),
				})
			if err != nil {
				_ = tx.Rollback()
				return errors.Wrap(err, "can't delete banner")
			}

			effected, err := res.RowsAffected()
			if err != nil {
				return errors.Wrap(err, "can't get result of deleting banner")
			}

			if effected == 0 {
				return repository.ErrorBannerNotFound
			}

			if err := br.runCronJob(); err != nil {
				return err
			}

			return nil
		},
	); err != nil {
		return errors.Wrapf(err,
			"when deleting banner with feature id %d or tag id %d", bnr.FeatureId.Value, bnr.TagId.Value)
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

	switch e.Code {
	case uniqueConflictCode:
		if e.ConstraintName == uniqueConstraintName {
			return repository.ErrorBannerConflictExists
		}
	}
	return err
}
