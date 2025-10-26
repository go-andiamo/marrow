package marrow

import "github.com/go-andiamo/marrow/framing"

type Runnable interface {
	Run(ctx Context) error
	framing.Framed
}
