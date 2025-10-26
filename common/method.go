package common

import "github.com/go-andiamo/marrow/framing"

type Method interface {
	MethodName() string
	Description() string
	framing.Framed
}
