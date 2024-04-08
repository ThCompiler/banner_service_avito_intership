package caches

import "bannersrv/internal/pkg/types"

type Manager interface {
	HaveCache(featureId types.Id, tagId types.Id, version *uint32) (types.Content, error)
	SetCache(featureId types.Id, tagId types.Id, version *uint32, content types.Content) error
}
