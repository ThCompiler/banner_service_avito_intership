package request

import (
	"bannersrv/internal/banner/models"
	"bannersrv/internal/pkg/evjson"
	"bannersrv/internal/pkg/types"
	"encoding/json"

	"github.com/miladibra10/vjson"
)

type CreateBanner struct {
	// Содержимое баннера
	Content json.RawMessage `json:"content" swaggertype:"object" additionalProperties:"true"`
	// Флаг активности баннера
	IsActive bool `json:"is_active" swaggertype:"boolean"`
	// Идентификатор фичи
	FeatureID types.ID `json:"feature_id" swaggertype:"integer" format:"uint64"`
	// Идентификаторы тегов
	TagsIDs []types.ID `json:"tag_ids"`
}

func ValidateCreateBanner(data []byte) error {
	schema := evjson.NewSchema(
		vjson.Object("content", vjson.NewSchema()).Required(),
		vjson.Boolean("is_active").Required(),
		vjson.Integer("feature_id").Positive().Required(),
		vjson.Array("tag_ids", vjson.Integer("id").Positive()).Required(),
	)

	return schema.ValidateBytes(data)
}

type UpdateBanner struct {
	// Содержимое баннера
	Content *json.RawMessage `json:"content,omitempty" swaggertype:"object" additionalProperties:"true"`
	// Флаг активности баннера
	IsActive *bool `json:"is_active,omitempty" swaggertype:"boolean"`
	// Идентификатор фичи
	FeatureID *types.ID `json:"feature_id,omitempty" swaggertype:"integer" format:"uint64"`
	// Идентификаторы тегов
	TagsIDs []types.ID `json:"tag_ids,omitempty"`
}

func ValidateUpdateBanner(data []byte) error {
	schema := evjson.NewSchema(
		vjson.Object("content", vjson.NewSchema()),
		vjson.Boolean("is_active"),
		vjson.Integer("feature_id").Positive(),
		vjson.Array("tag_ids", vjson.Integer("id").Positive()),
	)

	return schema.ValidateBytes(data)
}

func (ub *UpdateBanner) ToModel() *models.BannerUpdate {
	return &models.BannerUpdate{
		Content:   types.ObjectFromPointer(ub.Content),
		IsActive:  types.ObjectFromPointer(ub.IsActive),
		FeatureID: (*types.NullableID)(types.ObjectFromPointer(ub.FeatureID)),
		TagIDs: &types.NullableObject[[]types.ID]{
			IsNull: len(ub.TagsIDs) == 0,
			Value:  ub.TagsIDs,
		},
	}
}
