package caches

import "bannersrv/internal/pkg/types"

type Manager interface {
	HaveCache(featureId types.Id, tagId types.Id) (types.Content, error)
	SetCache(featureId types.Id, tagId types.Id, content types.Content) error
}
