package entity

import (
	"bannersrv/internal/pkg/types"
	"time"
)

type Content struct {
	Version   uint32
	Content   types.Content
	CreatedAt time.Time
}

type Banner struct {
	Id        types.Id
	FeatureId types.Id
	TagIds    []types.Id
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	Versions  []Content
}

type BannerUpdate struct {
	Id        types.Id
	Content   *types.NullableObject[types.Content]
	FeatureId *types.NullableObject[types.Id]
	TagIds    *types.NullableObject[[]types.Id]
	IsActive  *types.NullableObject[bool]
}

type BannerInfo struct {
	FeatureId *types.NullableObject[types.Id]
	TagId     *types.NullableObject[types.Id]
}
