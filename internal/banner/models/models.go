package models

import (
	"bannersrv/internal/banner/entity"
	"bannersrv/internal/pkg/types"
	"encoding/json"
	"time"
)

type Banner struct {
	Id        types.Id
	Content   json.RawMessage
	FeatureId types.Id
	TagIds    []types.Id
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type BannerUpdate struct {
	Content   *types.NullableObject[json.RawMessage]
	FeatureId *types.NullableObject[types.Id]
	TagIds    *types.NullableObject[[]types.Id]
	IsActive  *types.NullableObject[bool]
}

func FromBannerEntity(banner *entity.Banner) *Banner {
	return &Banner{
		Id:        banner.Id,
		Content:   json.RawMessage(banner.Content),
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
