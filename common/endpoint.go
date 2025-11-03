package common

import "github.com/go-andiamo/marrow/framing"

type Endpoint interface {
	// Url is the endpoint URL (including any ancestors)
	Url() string
	// Path is the endpoint path (i.e. url excluding ancestors)
	Path() string
	Description() string
	framing.Framed
}
