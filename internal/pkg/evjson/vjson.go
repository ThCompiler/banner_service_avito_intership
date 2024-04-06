package evjson

import (
	"github.com/miladibra10/vjson"
	"github.com/pkg/errors"
)

const jsonError = "could not parse json input."

var ErrorInvalidJson = errors.New(jsonError)

type Schema struct {
	vjson.Schema
}

func NewSchema(fields ...vjson.Field) Schema {
	return Schema{vjson.NewSchema(fields...)}
}

func (s *Schema) ValidateBytes(input []byte) error {
	if err := s.Schema.ValidateBytes(input); err != nil {
		if err.Error() == jsonError {
			return ErrorInvalidJson
		}
		return err
	}
	return nil
}

func (s *Schema) ValidateString(input string) error {
	if err := s.Schema.ValidateString(input); err != nil {
		if err.Error() == jsonError {
			return ErrorInvalidJson
		}
		return err
	}
	return nil
}
