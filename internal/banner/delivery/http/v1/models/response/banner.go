package response

import (
	"bannersrv/internal/banner/models"
	"bannersrv/internal/pkg/types"
	"encoding/json"
	"time"
)

type BannerId struct {
	// Идентификатор созданного баннера
	BannerId types.Id `json:"banner_id" swaggertype:"integer" format:"uint64"`
}

type Banner struct {
	// Идентификатор баннера
	Id types.Id `json:"banner_id" swaggertype:"integer" format:"uint64"`
	// Содержимое баннера
	Content json.RawMessage `json:"content" swaggertype:"object" additionalProperties:"true"`
	// Идентификатор фичи
	FeatureId types.Id `json:"feature_id" swaggertype:"integer"`
	// Идентификаторы тэгов
	TagIds []types.Id `json:"tag_ids"`
	// Флаг активности баннера
	IsActive bool `json:"is_active" swaggertype:"boolean" format:"uint64"`
	// Дата создания баннера
	CreatedAt time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
	// Дата обновления баннера
	UpdatedAt time.Time `json:"updated_at" swaggertype:"string" format:"date-time"`
}

func FromModelBanner(banner *models.Banner) *Banner {
	return &Banner{
		Id:        banner.Id,
		Content:   banner.Content,
		FeatureId: banner.FeatureId,
		TagIds:    banner.TagIds,
		IsActive:  banner.IsActive,
		CreatedAt: banner.CreatedAt,
		UpdatedAt: banner.UpdatedAt,
	}
}
