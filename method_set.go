package marrow

import (
	"errors"
	"fmt"
	"github.com/go-andiamo/marrow/framing"
)

//go:noinline
func SetQueryParam(name string, values ...any) Capture {
	return &methodSet{
		name: fmt.Sprintf("SetQueryParam(%q)", name),
		do: func(ctx Context, method Method_) (err error) {
			if ctx.CurrentRequest() == nil {
				method.QueryParam(name, values...)
			} else {
				err = errors.New("set query param too late - request already built")
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

//go:noinline
func SetRequestHeader(name string, value any) Capture {
	return &methodSet{
		name: fmt.Sprintf("SetRequestHeader(%q)", name),
		do: func(ctx Context, method Method_) (err error) {
			if ctx.CurrentRequest() == nil {
				method.RequestHeader(name, value)
			} else {
				err = errors.New("set request header too late - request already built")
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

//go:noinline
func SetRequestBody(value any) Capture {
	return &methodSet{
		name: "SetRequestBody()",
		do: func(ctx Context, method Method_) (err error) {
			if ctx.CurrentRequest() == nil {
				method.RequestBody(value)
			} else {
				err = errors.New("set request body too late - request already built")
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

//go:noinline
func SetRequestUseCookie(name string) Capture {
	return &methodSet{
		name: fmt.Sprintf("SetRequestUseCookie(%q)", name),
		do: func(ctx Context, method Method_) (err error) {
			if ctx.CurrentRequest() == nil {
				method.UseCookie(name)
			} else {
				err = errors.New("set request use cookie too late - request already built")
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

type methodSet struct {
	name  string
	do    func(ctx Context, method Method_) error
	frame *framing.Frame
}

var _ Capture = (*methodSet)(nil)

func (ms *methodSet) Name() string {
	return ms.name
}

func (ms *methodSet) Run(ctx Context) error {
	if m := ctx.CurrentMethod(); m != nil {
		return ms.do(ctx, m)
	} else {
		return fmt.Errorf("method not set yet: %s", ms.name)
	}
}

func (ms *methodSet) Frame() *framing.Frame {
	return ms.frame
}
