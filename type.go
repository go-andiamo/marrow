package marrow

import "reflect"

type Type_ interface {
	Type() reflect.Type
}

// Type is a generic func to determine a type (as used by Method.AssertType/Method.RequireType)
//
// example:
//
//	Method(GET, "").AssertType(Var("foo"), Type[string]())
//
// asserts that the resolved value of Var("foo") is of type string
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
