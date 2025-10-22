package marrow

import (
	"fmt"
	"reflect"
)

type Endpoint_ interface {
	Url() string
	Description() string
	Runnable
	fmt.Stringer
}

//go:noinline
func Endpoint(url string, desc string, operations ...any) Endpoint_ {
	result := &endpoint{
		url:   url,
		desc:  desc,
		frame: frame(0),
	}
	for _, o := range operations {
		if o == nil {
			continue
		}
		switch op := o.(type) {
		case Method_:
			result.methods = append(result.methods, op)
		case []Method_:
			for _, m := range op {
				if m == nil {
					continue
				}
				result.methods = append(result.methods, m)
			}
		case BeforeAfter_:
			if op.When() == Before {
				result.befores = append(result.befores, op)
			} else {
				result.afters = append(result.afters, op)
			}
		case []BeforeAfter_:
			for _, ba := range op {
				if ba == nil {
					continue
				}
				if ba.When() == Before {
					result.befores = append(result.befores, ba)
				} else {
					result.afters = append(result.afters, ba)
				}
			}
		default:
			if reflect.TypeOf(o).Kind() == reflect.Slice {
				vs := reflect.ValueOf(o)
				for i := 0; i < vs.Len(); i++ {
					v := vs.Index(i).Interface()
					switch opv := v.(type) {
					case Method_:
						result.methods = append(result.methods, opv)
					case BeforeAfter_:
						if opv.When() == Before {
							result.befores = append(result.befores, opv)
						} else {
							result.afters = append(result.afters, opv)
						}
					default:
						if v != nil {
							panic(fmt.Sprintf("unsupported operation type %T", o))
						}
					}
				}
			} else {
				panic(fmt.Sprintf("unsupported operation type %T", o))
			}
		}
	}
	return result
}

type endpoint struct {
	desc    string
	url     string
	frame   *Frame
	methods []Method_
	befores []BeforeAfter_
	afters  []BeforeAfter_
}

func (e *endpoint) String() string {
	return fmt.Sprintf("%s %q", e.url, e.desc)
}

func (e *endpoint) Url() string {
	return e.url
}

func (e *endpoint) Description() string {
	return e.desc
}

func (e *endpoint) Run(ctx Context) error {
	ctx.setCurrentEndpoint(e)
	for i, b := range e.befores {
		if !ctx.run(fmt.Sprintf("Before[%d]", i+1), b) {
			return nil
		}
	}
	for _, m := range e.methods {
		if !ctx.run("Method: "+string(m.Method())+" "+m.Description(), m) {
			return nil
		}
	}
	ctx.setCurrentMethod(nil)
	for i, a := range e.afters {
		if !ctx.run(fmt.Sprintf("After[%d]", i+1), a) {
			return nil
		}
	}
	return nil
}

func (e *endpoint) Frame() *Frame {
	return e.frame
}
