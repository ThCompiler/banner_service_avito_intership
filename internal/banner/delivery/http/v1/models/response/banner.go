package response

import (
	"bannersrv/internal/banner/models"
	"bannersrv/internal/pkg/types"
	"bannersrv/pkg/slices"
	"encoding/json"
	"time"
)

type BannerId struct {
	// Идентификатор созданного баннера
	BannerId types.Id `json:"banner_id" swaggertype:"integer" format:"uint64"`
}

type Content struct {
	// Содержимое баннера
	Content json.RawMessage `json:"content" swaggertype:"object" additionalProperties:"true"`
	// Версия содержимого баннера
	Version uint32 `json:"version" swaggertype:"integer" format:"uint32"`
	// Дата создания версии
	CreatedAt time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
}

type Banner struct {
	// Идентификатор баннера
	Id types.Id `json:"banner_id" swaggertype:"integer" format:"uint64"`
	// Последние три версии баннера
	Versions []Content `json:"versions"`
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

func FromModelContent(banner *models.Content) *Content {
	return &Content{
		Content:   banner.Content,
		Version:   banner.Version,
		CreatedAt: banner.CreatedAt,
	}
}

func FromModelBanner(banner *models.Banner) *Banner {
	return &Banner{
		Id: banner.Id,
		Versions: slices.Map(banner.Versions, func(content *models.Content) Content {
			return *FromModelContent(content)
		}),
		FeatureId: banner.FeatureId,
		TagIds:    banner.TagIds,
		IsActive:  banner.IsActive,
		CreatedAt: banner.CreatedAt,
		UpdatedAt: banner.UpdatedAt,
	}
}
