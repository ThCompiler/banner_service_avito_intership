package models

import (
	"bannersrv/internal/banner/entity"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/slices"
	"encoding/json"
	"time"
)

type Content struct {
	Version   uint32
	Content   json.RawMessage
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
	Content   *types.NullableObject[json.RawMessage]
	FeatureID *types.NullableObject[types.ID]
	TagIDs    *types.NullableObject[[]types.ID]
	IsActive  *types.NullableObject[bool]
}

func FromContentEntity(banner *entity.Content) *Content {
	return &Content{
		Content:   json.RawMessage(banner.Content),
		Version:   banner.Version,
		CreatedAt: banner.CreatedAt,
	}
}

func FromBannerEntity(banner *entity.Banner) *Banner {
	return &Banner{
		ID: banner.ID,
		Versions: slices.Map(banner.Versions, func(content *entity.Content) Content {
			return *FromContentEntity(content)
		}),
		FeatureID: banner.FeatureID,
		TagIDs:    banner.TagIDs,
		IsActive:  banner.IsActive,
		CreatedAt: banner.CreatedAt,
		UpdatedAt: banner.UpdatedAt,
	}
}

func (bu *BannerUpdate) ToBannerUpdateEntity(id types.ID) *entity.BannerUpdate {
	return &entity.BannerUpdate{
		ID: id,
		Content: &types.NullableObject[types.Content]{
			IsNull: bu.Content.IsNull,
			Value:  types.Content(bu.Content.Value),
		},
		FeatureID: bu.FeatureID,
		TagIDs:    bu.TagIDs,
		IsActive:  bu.IsActive,
	}
}
