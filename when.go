package marrow

import (
	"github.com/go-andiamo/marrow/framing"
)

// BeforeAfter is an interface that can be passed as an operation to Endpoint() or Method() (or Method.Capture)
// and instructs an operation to be run before or after the primary operations
type BeforeAfter interface {
	When() When
	Runnable
}

// When is a type that indicates Before or After
type When int

const (
	Before When = iota // an operation to run before primary operations (e.g. the actual method http request)
	After              // an operation to be run after primary operations (e.g. the actual method http request)
)

// DoAfter creates an after for the provided operation
//
// if the op is nil, the resulting before/after is nil
func DoAfter(op Runnable) BeforeAfter {
	if op == nil {
		return nil
	}
	return &beforeAfter{
		when: After,
		do:   op,
	}
}

// DoBefore creates a before for the provided operation
//
// if the op is nil, the resulting before/after is nil
func DoBefore(op Runnable) BeforeAfter {
	if op == nil {
		return nil
	}
	return &beforeAfter{
		when: Before,
		do:   op,
	}
}

type beforeAfter struct {
	when When
	do   Runnable
}

var _ BeforeAfter = (*beforeAfter)(nil)

func (b *beforeAfter) When() When {
	return b.when
}

func (b *beforeAfter) Run(ctx Context) error {
	return b.do.Run(ctx)
}

func (b *beforeAfter) Frame() *framing.Frame {
	return b.do.Frame()
}
