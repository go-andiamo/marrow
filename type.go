package marrow

import "reflect"

type Type_ interface {
	Type() reflect.Type
}

func Type[T any]() Type_ {
	var t T
	return &type_{
		to: reflect.TypeOf(t),
	}
}

type type_ struct {
	to reflect.Type
}

var _ Type_ = (*type_)(nil)

func (t *type_) Type() reflect.Type {
	return t.to
}
