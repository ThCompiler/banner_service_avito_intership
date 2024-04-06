package banner

import (
	"bannersrv/internal/banner/models"
	"bannersrv/internal/pkg/types"
	"encoding/json"
)

type Usecase interface {
	CreateBanner(tagIds []types.Id, featureId types.Id, content json.RawMessage, isActive bool) (types.Id, error)
	DeleteBanner(id types.Id) error
	UpdateBanner(id types.Id, banner *models.BannerUpdate) error
	GetAdminBanners(featureId *types.Id, tagId *types.Id, offset *uint64, limit *uint64) ([]models.Banner, error)
	GetUserBanner(featureId types.Id, tagId types.Id) (json.RawMessage, error)
}
