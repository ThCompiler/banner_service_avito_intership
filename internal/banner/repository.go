package banner

import (
	"bannersrv/internal/banner/entity"
	"bannersrv/internal/pkg/types"
)

type Repository interface {
	CreateBanner(featureId types.Id, tagIds []types.Id, content types.Content, isActive bool) (types.Id, error)
	DeleteBanner(id types.Id) (types.Id, error)
	UpdateBanner(banner *entity.BannerUpdate) (types.Id, error)
	GetBanners(banner *entity.BannerInfo, offset uint64, limit uint64) ([]entity.Banner, error)
	GetBanner(featureId types.Id, tagId types.Id, version types.NullableObject[uint32]) (types.Content, error)
	DeleteFilteredBanner(banner *entity.BannerInfo) error
}
