package banner

import (
	"bannersrv/internal/banner/entity"
	"bannersrv/internal/pkg/types"
)

type Repository interface {
	CreateBanner(featureID types.ID, tagIDs []types.ID, content types.Content, isActive bool) (types.ID, error)
	DeleteBanner(id types.ID) (types.ID, error)
	UpdateBanner(banner *entity.BannerUpdate) (types.ID, error)
	GetBanners(banner *entity.BannerInfo, offset, limit uint64) ([]entity.Banner, error)
	GetBanner(featureID, tagID types.ID, version types.NullableObject[uint32]) (types.Content, error)
	DeleteFilteredBanner(banner *entity.BannerInfo) error
	CleanDeletedBanner() error
}
