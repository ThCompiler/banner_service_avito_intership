package handlers

import "github.com/pkg/errors"

var (
	ErrorTagIdNotPresented      = errors.New("tag id not presented in query")
	ErrorFeatureIdNotPresented  = errors.New("feature id not presented in query")
	ErrorTagIdIncorrectType     = errors.New("tag id have incorrect type")
	ErrorFeatureIdIncorrectType = errors.New("feature id have incorrect type")

	ErrorLimitNotPresented   = errors.New("limit not presented in query")
	ErrorOffsetNotPresented  = errors.New("offset not presented in query")
	ErrorLimitIncorrectType  = errors.New("limit have incorrect type")
	ErrorOffsetIncorrectType = errors.New("offset have incorrect type")
)
