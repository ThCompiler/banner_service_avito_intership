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
	Id        types.Id
	FeatureId types.Id
	TagIds    []types.Id
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	Versions  []Content
}

type BannerUpdate struct {
	Content   *types.NullableObject[json.RawMessage]
	FeatureId *types.NullableObject[types.Id]
	TagIds    *types.NullableObject[[]types.Id]
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
		Id: banner.Id,
		Versions: slices.Map(banner.Versions, func(content *entity.Content) Content {
			return *FromContentEntity(content)
		}),
		FeatureId: banner.FeatureId,
		TagIds:    banner.TagIds,
		IsActive:  banner.IsActive,
		CreatedAt: banner.CreatedAt,
		UpdatedAt: banner.UpdatedAt,
	}
}

func (bu *BannerUpdate) ToBannerUpdateEntity(id types.Id) *entity.BannerUpdate {
	return &entity.BannerUpdate{
		Id: id,
		Content: &types.NullableObject[types.Content]{
			IsNull: bu.Content.IsNull,
			Value:  types.Content(bu.Content.Value),
		},
		FeatureId: bu.FeatureId,
		TagIds:    bu.TagIds,
		IsActive:  bu.IsActive,
	}
}
