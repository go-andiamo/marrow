package marrow

import (
	"github.com/go-andiamo/marrow/framing"
	"strings"
)

type BeforeAfter_ interface {
	When() When
	Runnable
}

type When int

const (
	Before When = iota
	After
)

type beforeAfter struct {
	when When
	do   Runnable
}

var _ BeforeAfter_ = (*beforeAfter)(nil)

func (b *beforeAfter) When() When {
	return b.when
}

func (b *beforeAfter) Run(ctx Context) error {
	return b.do.Run(ctx)
}

func (b *beforeAfter) Frame() *framing.Frame {
	return b.do.Frame()
}

//go:noinline
func SetVar(when When, name string, value any) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &setVar{
			name:  name,
			value: value,
			frame: framing.NewFrame(0),
		},
	}
}

//go:noinline
func ClearVars(when When) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &clearVars{
			frame: framing.NewFrame(0),
		},
	}
}

//go:noinline
func DbInsert(when When, tableName string, row Columns) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &dbInsert{
			tableName: tableName,
			row:       row,
			frame:     framing.NewFrame(0),
		},
	}
}

//go:noinline
func DbExec(when When, query string, args ...any) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &dbExec{
			query: query,
			args:  args,
			frame: framing.NewFrame(0),
		},
	}
}

//go:noinline
func DbClearTable(when When, tableName string) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &dbClearTable{
			tableName: tableName,
			frame:     framing.NewFrame(0),
		},
	}
}

//go:noinline
func MockServicesClearAll(when When) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &mockServicesClearAll{
			frame: framing.NewFrame(0),
		},
	}
}

//go:noinline
func MockServiceClear(when When, svcName string) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &mockServiceClear{
			name:  svcName,
			frame: framing.NewFrame(0),
		},
	}
}

//go:noinline
func MockServiceCall(when When, svcName string, path string, method MethodName, responseStatus int, responseBody any, headers ...any) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &mockServiceCall{
			name:           svcName,
			path:           path,
			method:         strings.ToUpper(string(method)),
			responseStatus: responseStatus,
			responseBody:   responseBody,
			headers:        headers,
			frame:          framing.NewFrame(0),
		},
	}
}
