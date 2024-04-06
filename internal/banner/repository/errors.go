package repository

import "github.com/pkg/errors"

var (
	ErrorBannerNotFound       = errors.New("banner not found")
	ErrorBannerConflictExists = errors.New("banner with presented pair featured id and tag id is already exists")
)
