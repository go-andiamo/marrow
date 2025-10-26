package common

import "github.com/go-andiamo/marrow/framing"

type Endpoint interface {
	Url() string
	Description() string
	framing.Framed
}
