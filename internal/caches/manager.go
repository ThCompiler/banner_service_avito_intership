package caches

import "bannersrv/internal/pkg/types"

type Manager interface {
	HaveCache(featureID, tagID types.ID, version *uint32) (types.Content, error)
	SetCache(featureID, tagID types.ID, version *uint32, content types.Content) error
}
