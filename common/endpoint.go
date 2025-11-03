package common

import "github.com/go-andiamo/marrow/framing"

// Endpoint is the interface that describes an API endpoint (exc. method)
//
// Multiple Endpoint interfaces can be passed to marrow.Suite constructor to describe the endpoints
// to be tested
type Endpoint interface {
	// Url is the endpoint URL (including any ancestors)
	Url() string
	// Path is the endpoint path (i.e. url excluding ancestors)
	Path() string
	Description() string
	framing.Framed
}
