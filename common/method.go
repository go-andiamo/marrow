package common

import "github.com/go-andiamo/marrow/framing"

// Method is the interface that describes a http method on an Endpoint
type Method interface {
	MethodName() string // typically returns "GET", "POST", "DELETE" etc.
	Description() string
	framing.Framed
}
