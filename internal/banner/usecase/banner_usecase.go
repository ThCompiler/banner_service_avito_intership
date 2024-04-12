package usecase

import (
	"bannersrv/internal/banner"
	"bannersrv/internal/banner/entity"
	"bannersrv/internal/banner/models"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/slices"
	"encoding/json"
)

const (
	defaultOffset = 0
	defaultLimit  = 100
)

type BannerUsecase struct {
	rep banner.Repository
}

func NewBannerUsecase(bnr banner.Repository) *BannerUsecase {
	return &BannerUsecase{
		rep: bnr,
	}
}

func (bu *BannerUsecase) CreateBanner(tagIDs []types.ID, featureID types.ID,
	content json.RawMessage, isActive bool,
) (types.ID, error) {
	return bu.rep.CreateBanner(featureID, tagIDs, types.Content(content), isActive)
}

func (bu *BannerUsecase) DeleteBanner(id types.ID) error {
	_, err := bu.rep.DeleteBanner(id)

	return err
}

func (bu *BannerUsecase) UpdateBanner(id types.ID, bnr *models.BannerUpdate) error {
	_, err := bu.rep.UpdateBanner(bnr.ToBannerUpdateEntity(id))

	return err
}

func (bu *BannerUsecase) GetAdminBanners(featureID, tagID *types.ID,
	offset, limit *uint64,
) ([]models.Banner, error) {
	var entityOffset uint64 = defaultOffset

	var entityLimit uint64 = defaultLimit

	if offset != nil {
		entityOffset = *offset
	}

	if limit != nil {
		entityLimit = *limit
	}

	banners, err := bu.rep.GetBanners(&entity.BannerInfo{
		FeatureID: (*types.NullableID)(types.ObjectFromPointer(featureID)),
		TagID:     (*types.NullableID)(types.ObjectFromPointer(tagID)),
	}, entityOffset, entityLimit)
	if err != nil {
		return nil, err
	}

	return slices.Map(banners, func(b *entity.Banner) models.Banner {
		return *models.FromBannerEntity(b)
	}), nil
}

func (bu *BannerUsecase) GetUserBanner(featureID, tagID types.ID, version *uint32) (json.RawMessage, error) {
	content, err := bu.rep.GetBanner(featureID, tagID, *types.ObjectFromPointer(version))
	if err != nil {
		return nil, err
	}

	return json.RawMessage(content), nil
}

func (bu *BannerUsecase) DeleteFilteredBanner(featureID, tagID *types.ID) error {
	return bu.rep.DeleteFilteredBanner(&entity.BannerInfo{
		FeatureID: (*types.NullableID)(types.ObjectFromPointer(featureID)),
		TagID:     (*types.NullableID)(types.ObjectFromPointer(tagID)),
	})
}
