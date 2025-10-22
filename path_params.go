package marrow

import (
	"fmt"
	"github.com/go-andiamo/urit"
)

type pathParams []any

var _ urit.PathVars = pathParams{}

func (p pathParams) resolve(ctx Context) (result pathParams, err error) {
	result = make(pathParams, len(p))
	for i, v := range p {
		var av any
		if av, err = ResolveValue(v, ctx); err == nil {
			result[i] = av
		} else {
			return nil, err
		}
	}
	return result, nil
}

func (p pathParams) GetPositional(position int) (string, bool) {
	if position >= 0 && position < len(p) {
		v := p[position]
		switch t := v.(type) {
		case string:
			return t, true
		default:
			return fmt.Sprintf("%v", v), true
		}
	}
	return "", false
}

func (p pathParams) GetNamed(name string, position int) (string, bool) {
	panic("not implemented, not used")
}

func (p pathParams) GetNamedFirst(name string) (string, bool) {
	panic("not implemented, not used")
}

func (p pathParams) GetNamedLast(name string) (string, bool) {
	panic("not implemented, not used")
}

func (p pathParams) Get(idents ...interface{}) (string, bool) {
	panic("not implemented, not used")
}

func (p pathParams) GetAll() []urit.PathVar {
	panic("not implemented, not used")
}

func (p pathParams) Len() int {
	return len(p)
}

func (p pathParams) Clear() {
	panic("not implemented, not used")
}

func (p pathParams) VarsType() urit.PathVarsType {
	return urit.Positions
}

func (p pathParams) AddNamedValue(name string, val interface{}) error {
	panic("not implemented, not used")
}

func (p pathParams) AddPositionalValue(val interface{}) error {
	panic("not implemented, not used")
}
