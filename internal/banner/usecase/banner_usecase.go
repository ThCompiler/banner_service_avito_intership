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

func NewBannerUsecase(banner banner.Repository) *BannerUsecase {
	return &BannerUsecase{
		rep: banner,
	}
}

func (bu *BannerUsecase) CreateBanner(tagIds []types.Id, featureId types.Id,
	content json.RawMessage, isActive bool) (types.Id, error) {
	return bu.rep.CreateBanner(&entity.Banner{
		TagIds:    tagIds,
		FeatureId: featureId,
		Content:   types.Content(content),
		IsActive:  isActive,
	})
}

func (bu *BannerUsecase) DeleteBanner(id types.Id) error {
	_, err := bu.rep.DeleteBanner(id)
	return err
}

func (bu *BannerUsecase) UpdateBanner(id types.Id, banner *models.BannerUpdate) error {
	_, err := bu.rep.UpdateBanner(banner.ToBannerUpdateEntity(id))
	return err
}

func (bu *BannerUsecase) GetAdminBanners(featureId *types.Id, tagId *types.Id,
	offset *uint64, limit *uint64) ([]models.Banner, error) {
	var entityOffset uint64 = defaultOffset
	var entityLimit uint64 = defaultLimit

	if offset != nil {
		entityOffset = *offset
	}

	if limit != nil {
		entityLimit = *limit
	}

	banners, err := bu.rep.GetBanners(&entity.BannerInfo{
		FeatureId: types.ObjectFromPointer(featureId),
		TagId:     types.ObjectFromPointer(tagId),
	}, entityOffset, entityLimit)
	if err != nil {
		return nil, err
	}

	return slices.Map(banners, func(b entity.Banner) models.Banner {
		return *models.FromBannerEntity(&b)
	}), nil
}

func (bu *BannerUsecase) GetUserBanner(featureId types.Id, tagId types.Id) (json.RawMessage, error) {
	content, err := bu.rep.GetBanner(featureId, tagId)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(content), nil
}
