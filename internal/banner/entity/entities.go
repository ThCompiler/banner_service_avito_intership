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
	ID        types.ID
	FeatureID types.ID
	TagIDs    []types.ID
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	Versions  []Content
}

type BannerUpdate struct {
	ID        types.ID
	Content   *types.NullableObject[types.Content]
	FeatureID *types.NullableID
	TagIDs    *types.NullableObject[[]types.ID]
	IsActive  *types.NullableObject[bool]
}

type BannerInfo struct {
	FeatureID *types.NullableID
	TagID     *types.NullableID
}
