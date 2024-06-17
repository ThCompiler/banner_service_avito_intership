package banner

import (
	"bannersrv/internal/banner/models"
	"bannersrv/internal/pkg/types"
	"encoding/json"
)

type Usecase interface {
	CreateBanner(tagIDs []types.ID, featureID types.ID, content json.RawMessage, isActive bool) (types.ID, error)
	DeleteBanner(id types.ID) error
	UpdateBanner(id types.ID, banner *models.BannerUpdate) error
	GetAdminBanners(featureID, tagID *types.ID, offset, limit *uint64) ([]models.Banner, error)
	GetUserBanner(featureID, tagID types.ID, version *uint32) (json.RawMessage, error)
	DeleteFilteredBanner(featureID, tagID *types.ID) error
}
