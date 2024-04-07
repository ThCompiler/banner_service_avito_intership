package types

import (
	"database/sql"
)

type Id uint32

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

func (object *NullableObject[T]) ToNullableSQL() *sql.Null[T] {
	if object == nil {
		return &sql.Null[T]{
			Valid: false,
		}
	}
	return &sql.Null[T]{
		Valid: !object.IsNull,
		V:     object.Value,
	}
}
