package postgres

import (
	"bannersrv/internal/banner/entity"
	"bannersrv/internal/banner/repository"
	"bannersrv/internal/pkg/pg"
	"bannersrv/internal/pkg/types"
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
		DELETE FROM banner WHERE id IN 
		 (SELECT DISTINCT banner_id FROM features_tags_banner 
				  WHERE not deleted and banner_id = $1) RETURNING id
	`

	checkDeleted = `
		SELECT banner_id FROM features_tags_banner WHERE banner_id = $1 and not deleted
	`

	updateActiveQuery = `
		UPDATE banner SET is_active = $2
			WHERE id = $1
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
		   INNER JOIN features_tags_banner on (features_tags_banner.banner_id = banner.id and not deleted)
		   LEFT JOIN version_banner as vb on (vb.banner_id = banner.id)
		WHERE is_active and vb.version = COALESCE($3::bigint, banner.last_version) 
		  		and feature_id = $1 and tag_id = $2 LIMIT 1
	`

	filterNullQuery = `
		SELECT banner.id, is_active, created_at, updated_at FROM banner LIMIT $1 OFFSET $2
	`

	filterNotNullQuery = `
		SELECT DISTINCT banner.id, is_active,
                        created_at, updated_at FROM banner
			INNER JOIN features_tags_banner as ftb ON (ftb.banner_id = banner.id  and not deleted)
			WHERE (CASE WHEN $1::bigint IS NOT NULL THEN feature_id = $1 ELSE true END)
			and (CASE WHEN $2::bigint IS NOT NULL THEN tag_id = $2 ELSE true END)
			LIMIT $3 OFFSET $4
	`

	getTagQuery = `
		SELECT banner_id, array_agg(tag_id), feature_id FROM features_tags_banner 
		                                                WHERE banner_id = ANY ($1::bigint[])
		GROUP BY banner_id, feature_id 
	`

	getVersionQuery = `
		SELECT banner_id, content, version, created_at FROM version_banner WHERE banner_id = ANY ($1::bigint[])
	`

	delayedDeletionQuery = `
		UPDATE features_tags_banner SET deleted = true 
			 WHERE (CASE WHEN $1::bigint IS NOT NULL THEN feature_id = $1 ELSE true END)
					and (CASE WHEN $2::bigint IS NOT NULL THEN tag_id = $2 ELSE true END) 
	`

	cronDeleteQuery = `
		DELETE FROM banner WHERE id IN (SELECT DISTINCT banner_id FROM features_tags_banner WHERE not deleted)
	`
)

type BannerRepository struct {
	db *pgxpool.Pool
}

func NewBannerRepository(db *pgxpool.Pool) *BannerRepository {
	return &BannerRepository{
		db: db,
	}
}

func (*BannerRepository) addContent(tx pgx.Tx, id types.ID, content types.Content) error {
	if _, err := tx.Exec(context.Background(), addContentQuery, id, content); err != nil {
		return errors.Wrap(err, "can't add content to banner")
	}

	return nil
}

func (br *BannerRepository) CreateBanner(featureID types.ID, tagIDs []types.ID,
	content types.Content, isActive bool,
) (types.ID, error) {
	var createdID types.ID

	if err := pg.WithTransaction(br.db,
		func(tx pgx.Tx) error {
			if err := tx.QueryRow(context.Background(), createQuery, isActive).
				Scan(
					&createdID,
				); err != nil {
				return errors.Wrap(err, "can't create banner")
			}

			if err := br.addContent(tx, createdID, content); err != nil {
				return err
			}

			if _, err := tx.Exec(context.Background(), addFeaturesAndTagsQuery, createdID, featureID,
				pgtype.FlatArray[types.ID](tagIDs)); err != nil {
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
	if err := br.db.QueryRow(context.Background(), deleteQuery, id).
		Scan(
			&deletedID,
		); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return deletedID, errors.Wrapf(repository.ErrorBannerNotFound, "with id %d", id)
		}

		return deletedID, errors.Wrapf(err, "can't delete banner with id %d", id)
	}

	return deletedID, nil
}

func (*BannerRepository) updateBannerInfo(tx pgx.Tx, bnr *entity.BannerUpdate) error {
	switch {
	// Если у нас изменился только айди фичи, её можно обновить по id баннера
	case bnr.TagIDs.IsNull && !bnr.FeatureID.IsNull:
		if _, err := tx.Exec(context.Background(), updateFeaturesQuery, bnr.ID, bnr.FeatureID.Value); err != nil {
			return errors.Wrapf(checkPgConflictError(err),
				"can't update feature id %d to banner", bnr.FeatureID.Value)
		}
	// Если у нас изменился список тэгов, то нужно сначала удалить все записи с тэгами, а потом их снова создать
	case !bnr.TagIDs.IsNull:
		var featureID types.ID
		if err := tx.QueryRow(context.Background(), deleteFeaturesTagsQuery, bnr.ID).Scan(&featureID); err != nil {
			return errors.Wrap(err, "can't delete feature id and tag ids of banner")
		}

		if !bnr.FeatureID.IsNull {
			featureID = bnr.FeatureID.Value
		}

		if _, err := tx.Exec(context.Background(), addFeaturesAndTagsQuery, bnr.ID, featureID,
			pgtype.FlatArray[types.ID](bnr.TagIDs.Value)); err != nil {
			return errors.Wrapf(checkPgConflictError(err),
				"can't add feature id %d and tag ids %v to banner", featureID, bnr.TagIDs.Value)
		}
	}

	return nil
}

func (br *BannerRepository) UpdateBanner(bnr *entity.BannerUpdate) (types.ID, error) {
	var updatedID types.ID

	if err := pg.WithTransaction(br.db,
		func(tx pgx.Tx) error {
			if err := tx.QueryRow(context.Background(), checkDeleted, bnr.ID).Scan(&updatedID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return repository.ErrorBannerNotFound
				}

				return errors.Wrapf(err, "can't check banner on deleted")
			}

			if !bnr.IsActive.IsNull {
				if err := tx.QueryRow(context.Background(), updateActiveQuery,
					bnr.ID, bnr.IsActive.Value).
					Scan(&updatedID); err != nil {
					if errors.Is(err, pgx.ErrNoRows) {
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

func (*BannerRepository) filterBanners(tx pgx.Tx, bnr *entity.BannerInfo,
	offset, limit uint64,
) ([]entity.Banner, error) {
	args := []any{bnr.FeatureID.ToNullableSQL(), bnr.TagID.ToNullableSQL(), limit, offset}
	query := filterNotNullQuery

	if bnr.FeatureID.IsNull && bnr.TagID.IsNull {
		args = []any{limit, offset}
		query = filterNullQuery
	}

	rows, err := tx.Query(context.Background(), query, args...)
	defer rows.Close() // nolint: staticcheck // Close() doesn't return error

	if err != nil {
		return nil, errors.Wrap(err, "can't execute filter banner query")
	}

	banners := make([]entity.Banner, 0)

	for rows.Next() {
		var filteredBanner entity.Banner

		err := rows.Scan(
			&filteredBanner.ID,
			&filteredBanner.IsActive,
			&filteredBanner.CreatedAt,
			&filteredBanner.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "can't scan filter banner query result")
		}

		filteredBanner.TagIDs = make([]types.ID, 0)
		filteredBanner.Versions = make([]entity.Content, 0)

		banners = append(banners, filteredBanner)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "can't end scan filter banner query result")
	}

	return banners, nil
}

func (*BannerRepository) selectTagFeatureForBanners(tx pgx.Tx, banners []entity.Banner) ([]entity.Banner, error) {
	bannerIDs := make([]types.ID, len(banners))
	bannerIndexes := make(map[types.ID]int64)

	for index, bnr := range banners {
		bannerIDs[index] = bnr.ID
		bannerIndexes[bnr.ID] = int64(index)
	}

	rows, err := tx.Query(context.Background(), getTagQuery, bannerIDs)
	defer rows.Close() // nolint: staticcheck // Close() doesn't return error

	if err != nil {
		return nil, errors.Wrap(err, "can't execute get tags for banner query")
	}

	for rows.Next() {
		var tags pgtype.Array[types.ID]

		var bannerID, featureID types.ID

		err := rows.Scan(
			&bannerID,
			&tags,
			&featureID,
		)
		if err != nil {
			return nil, errors.Wrap(err, "can't scan get tags for banner query result")
		}

		banners[bannerIndexes[bannerID]].FeatureID = featureID

		banners[bannerIndexes[bannerID]].TagIDs = []types.ID{}
		if tags.Valid {
			banners[bannerIndexes[bannerID]].TagIDs = tags.Elements
		}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "can't end scan get tags for banner query result")
	}

	return banners, nil
}

func (*BannerRepository) selectContentForBanners(tx pgx.Tx, banners []entity.Banner) ([]entity.Banner, error) {
	bannerIDs := make([]types.ID, len(banners))
	bannerIndexes := make(map[types.ID]int64)

	for index, bnr := range banners {
		bannerIDs[index] = bnr.ID
		bannerIndexes[bnr.ID] = int64(index)
	}

	rows, err := tx.Query(context.Background(), getVersionQuery, bannerIDs)
	defer rows.Close() // nolint: staticcheck // Close() doesn't return error

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
		func(tx pgx.Tx) error {
			var err error

			banners, err = br.filterBanners(tx, bnr, offset, limit)
			if err != nil {
				return err
			}

			if len(banners) == 0 {
				return nil
			}

			banners, err = br.selectTagFeatureForBanners(tx, banners)
			if err != nil {
				return err
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
	if err := br.db.QueryRow(context.Background(), getQuery, featureID, tagID,
		&pgtype.Uint32{
			Valid:  !version.IsNull,
			Uint32: version.Value,
		}).
		Scan(
			&content,
		); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return content, errors.Wrapf(repository.ErrorBannerNotFound,
				"with feature id %d and tag id %d and version %v", featureID, tagID, version)
		}

		return content, errors.Wrapf(err,
			"can't get banner with feature id %d and tag id %d and version %v", featureID, tagID, version)
	}

	return content, nil
}

func (br *BannerRepository) DeleteFilteredBanner(bnr *entity.BannerInfo) error {
	if err := pg.WithTransaction(br.db,
		func(tx pgx.Tx) error {
			res, err := tx.Exec(context.Background(), delayedDeletionQuery,
				&pgtype.Uint32{
					Valid:  !bnr.FeatureID.IsNull,
					Uint32: uint32(bnr.FeatureID.Value),
				}, &pgtype.Uint32{
					Valid:  !bnr.TagID.IsNull,
					Uint32: uint32(bnr.TagID.Value),
				})
			if err != nil {
				return errors.Wrap(err, "can't delete banner")
			}

			effected := res.RowsAffected()

			if effected == 0 {
				return repository.ErrorBannerNotFound
			}

			return nil
		},
	); err != nil {
		return errors.Wrapf(err,
			"when deleting banner with feature id %d or tag id %d", bnr.FeatureID.Value, bnr.TagID.Value)
	}

	return nil
}

func (br *BannerRepository) CleanDeletedBanner() error {
	_, err := br.db.Exec(context.Background(), cronDeleteQuery)
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
