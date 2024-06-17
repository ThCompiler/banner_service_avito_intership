package middleware

import "github.com/pkg/errors"

var ErrorUseLastRevisionIncorrectType = errors.New("use_last_revision param have incorrect type")
