package handlers

import "github.com/pkg/errors"

var (
	ErrorTagIDNotPresented      = errors.New("tag id not presented in query")
	ErrorFeatureIDNotPresented  = errors.New("feature id not presented in query")
	ErrorTagIDIncorrectType     = errors.New("tag id have incorrect type")
	ErrorFeatureIDIncorrectType = errors.New("feature id have incorrect type")

	ErrorLimitIncorrectType   = errors.New("limit have incorrect type")
	ErrorOffsetIncorrectType  = errors.New("offset have incorrect type")
	ErrorVersionIncorrectType = errors.New("version have incorrect type")

	ErrorParamsNotPresented = errors.New("feature id and tag id not presented in query")
)
