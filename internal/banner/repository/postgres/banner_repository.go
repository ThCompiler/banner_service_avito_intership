package postgres

import (
	"bannersrv/internal/banner/entity"
	"bannersrv/internal/banner/repository"
	"bannersrv/internal/pkg/types"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

const (
	createQuery = `
		INSERT INTO banner (content, is_active)
		VALUES ($1, $2)
		RETURNING id
	`

	addFeaturesAndTagsQuery = `
		INSERT INTO features_tags_banner (banner_id, feature_id, tag_id)
		SELECT $1, $2, tag
		FROM unnest($3::bigint[]) as tag		
	`

	deleteQuery = `
		DELETE FROM banner WHERE id = $1 RETURNING id
	`

	updateQuery = `
		UPDATE banner SET content = upd_banner.upd_content, is_active = upd_banner.upd_is_active
			FROM (
				SELECT COALESCE($2, banner.content) as upd_content, 
					   COALESCE($3, banner.is_active) as upd_is_active
				FROM banner WHERE id = $1
			) as upd_banner
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
		WITH selected_banner AS (
			SELECT banner_id FROM features_tags_banner WHERE feature_id = $1 and tag_id = $2
		) 
		SELECT content FROM banner INNER JOIN selected_banner on (selected_banner.banner_id = banner.id)
		WHERE is_active LIMIT 1
	`

	filterQuery = `
		WITH selected_banner AS (
			SELECT banner_id, feature_id, array_agg(tag_id) as tag_ids FROM features_tags_banner 
			WHERE (CASE WHEN $1 IS NOT NULL THEN feature_id = $1 ELSE true END)
      			and (CASE WHEN $2 IS NOT NULL THEN tag_id = $2 ELSE true END) 
			GROUP BY banner_id, feature_id
		) 
		SELECT id, tag_ids, feature_id, content, is_active, created_at, updated_at FROM banner 
		    INNER JOIN selected_banner ON (selected_banner.banner_id = banner.id)
			LIMIT $3 OFFSET $4
	`
)

type BannerRepository struct {
	db *sqlx.DB
}

func NewBannerRepository(db *sqlx.DB) *BannerRepository {
	return &BannerRepository{
		db: db,
	}
}

func (br *BannerRepository) CreateBanner(banner *entity.Banner) (types.Id, error) {
	tx, err := br.db.Beginx()
	if err != nil {
		return 0, errors.Wrap(err, "can't begin transaction to create banner")
	}

	var createdId types.Id
	if err := tx.QueryRowx(createQuery, banner.Content, banner.IsActive).
		Scan(
			&createdId,
		); err != nil {
		_ = tx.Rollback()
		return 0, errors.Wrap(err, "can't create banner")
	}

	if _, err := tx.Exec(addFeaturesAndTagsQuery, createdId, banner.FeatureId, banner.TagIds); err != nil {
		_ = tx.Rollback()
		return 0, errors.Wrapf(checkPgConflictError(err),
			"can't add feature id %d and tag ids %v to banner", banner.FeatureId, banner.TagIds)
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrap(err, "can't commit transaction to create banner")
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

func (br *BannerRepository) UpdateBanner(banner *entity.BannerUpdate) (types.Id, error) {
	tx, err := br.db.Beginx()
	if err != nil {
		return 0, errors.Wrapf(err, "can't begin transaction to update banner with id %d", banner.Id)
	}

	var updatedId types.Id
	if err := tx.QueryRowx(updateQuery, banner.Id, banner.Content.ToNullableSQL(), banner.IsActive.ToNullableSQL()).
		Scan(
			&updatedId,
		); err != nil {
		_ = tx.Rollback()
		return 0, errors.Wrapf(err, "can't create banner with id %d", banner.Id)
	}

	switch {
	// Если у нас изменился только айди фичи, её можно обновить по id баннера
	case banner.TagIds.IsNull && !banner.FeatureId.IsNull:
		if _, err := tx.Exec(updateFeaturesQuery, banner.Id, banner.FeatureId.Value); err != nil {
			_ = tx.Rollback()
			return 0, errors.Wrapf(checkPgConflictError(err),
				"can't update feature id %d to banner with id %d", banner.FeatureId.Value, banner.Id)
		}
	// Если у нас изменился список тэгов, то нужно сначала удалить все записи с тэгами, а потом их снова создать
	case !banner.TagIds.IsNull:
		var featureId types.Id
		if err := tx.QueryRowx(deleteFeaturesTagsQuery, banner.Id).Scan(&featureId); err != nil {
			_ = tx.Rollback()
			return 0, errors.Wrapf(err,
				"can't delete feature id and tag ids of banner with id %d", banner.Id)
		}

		if !banner.FeatureId.IsNull {
			featureId = banner.FeatureId.Value
		}

		if _, err := tx.Exec(addFeaturesAndTagsQuery, banner.Id, featureId, banner.TagIds.Value); err != nil {
			_ = tx.Rollback()
			return 0, errors.Wrapf(checkPgConflictError(err),
				"can't add feature id %d and tag ids %v to banner with id %d",
				featureId, banner.TagIds.Value, banner.Id)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, errors.Wrapf(err, "can't commit transaction to update banner with id %d", banner.Id)
	}

	return updatedId, nil
}

func (br *BannerRepository) GetBanners(banner *entity.BannerInfo,
	offset uint64, limit uint64) ([]entity.Banner, error) {
	rows, err := br.db.Queryx(filterQuery, banner.FeatureId.ToNullableSQL(), banner.TagIds.ToNullableSQL(),
		limit, offset)
	if err != nil {
		return nil, errors.Wrap(err, "can't execute filter banner query")
	}

	banners := make([]entity.Banner, 0)

	for rows.Next() {
		var banner entity.Banner

		err := rows.Scan(
			&banner.Id,
			&banner.TagIds,
			&banner.FeatureId,
			&banner.Content,
			&banner.IsActive,
			&banner.CreatedAt,
			&banner.UpdatedAt,
		)
		if err != nil {
			return nil, errors.Wrap(err, "can't scan filter banner query result")
		}

		banners = append(banners, banner)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "can't end scan filter banner query result")
	}

	return banners, nil
}

func (br *BannerRepository) GetBanner(featureId types.Id, tagId types.Id) (types.Content, error) {
	var content types.Content
	if err := br.db.QueryRowx(getQuery, featureId, tagId).
		Scan(
			&content,
		); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return content, errors.Wrapf(repository.ErrorBannerNotFound,
				"with feature id %d and tag id %d", featureId, tagId)
		}
		return content, errors.Wrapf(err, "can't get banner with feature id %d and tag id %d", featureId, tagId)
	}

	return content, nil
}

const (
	uniqueConflictCode   = "23505"
	uniqueConstraintName = "banner_identifier"
)

func checkPgConflictError(err error) error {
	var e *pq.Error

	if !errors.As(err, &e) {
		return err
	}

	switch e.Code {
	case uniqueConflictCode:
		if e.Constraint == uniqueConstraintName {
			return repository.ErrorBannerConflictExists
		}
	}
	return err
}
