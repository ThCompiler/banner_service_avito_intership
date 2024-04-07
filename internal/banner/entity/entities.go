package entity

import (
	"bannersrv/internal/pkg/types"
	"time"
)

type Banner struct {
	Id        types.Id
	Content   types.Content
	FeatureId types.Id
	TagIds    []types.Id
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
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
