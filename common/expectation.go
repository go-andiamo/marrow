package common

import "github.com/go-andiamo/marrow/framing"

type Expectation interface {
	Name() string
	framing.Framed
}
