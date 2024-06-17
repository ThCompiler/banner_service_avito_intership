package types

import (
	"database/sql/driver"
	"math"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pkg/errors"
)

var (
	ErrorCanNotScan     = errors.New("can not scan")
	ErrorLessMinimum    = errors.New("less than the minimum value for ID")
	ErrorGreaterMaximum = errors.New("greater than the maximum value for ID")
)

type ID uint32

type ContextField string

type Content string

type NullableObject[T any] struct {
	IsNull bool
	Value  T
}

func NewObject[T any](object T) *NullableObject[T] {
	return &NullableObject[T]{
		IsNull: false,
		Value:  object,
	}
}

func NewNullObject[T any]() *NullableObject[T] {
	return &NullableObject[T]{
		IsNull: true,
	}
}

func ObjectFromPointer[T any](object *T) *NullableObject[T] {
	if object == nil {
		return NewNullObject[T]()
	}

	return NewObject(*object)
}

type NullableID NullableObject[ID]

func (object *NullableID) ToNullableSQL() *pgtype.Uint32 {
	if object == nil {
		return &pgtype.Uint32{
			Valid: false,
		}
	}

	return &pgtype.Uint32{
		Valid:  !object.IsNull,
		Uint32: uint32(object.Value),
	}
}

func (content *Content) Scan(src any) error {
	if src == nil {
		*content = ""

		return nil
	}

	switch typed := src.(type) {
	case string:
		*content = Content(typed)

		return nil
	case []byte:
		*content = Content(typed)

		return nil
	}

	return errors.Wrapf(ErrorCanNotScan, "with type %T", src)
}

func (content Content) Value() (driver.Value, error) {
	return string(content), nil
}

func (id *ID) Scan(src any) error {
	if src == nil {
		*id = 0

		return nil
	}

	var n int64

	switch typed := src.(type) {
	case int64:
		n = typed
	case string:
		un, err := strconv.ParseUint(typed, 10, 32)
		if err != nil {
			return err
		}

		n = int64(un)
	default:
		return errors.Wrapf(ErrorCanNotScan, "with type %T", src)
	}

	if n < 0 {
		return errors.Wrapf(ErrorLessMinimum, "for value %d", n)
	}

	if n > math.MaxUint32 {
		return errors.Wrapf(ErrorGreaterMaximum, "for value %d", n)
	}

	*id = ID(n)

	return nil
}

func (id ID) Value() (driver.Value, error) {
	return int64(id), nil
}
