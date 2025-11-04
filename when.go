package marrow

import (
	"github.com/go-andiamo/marrow/framing"
	"strings"
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
	Before When = iota // an operation to run before primary operations
	After              // an operation to be run after primary operations
)

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

// SetVar is a before/after operation to set a variable in the current Context
//
//go:noinline
func SetVar(when When, name string, value any) BeforeAfter {
	return &beforeAfter{
		when: when,
		do: &setVar{
			name:  name,
			value: value,
			frame: framing.NewFrame(0),
		},
	}
}

// ClearVars is a before/after operation to clear all variables in the current Context
//
//go:noinline
func ClearVars(when When) BeforeAfter {
	return &beforeAfter{
		when: when,
		do: &clearVars{
			frame: framing.NewFrame(0),
		},
	}
}

// DbInsert is a before/after operation to insert into a database table
//
// Note: when only one database is used by tests, the dbName can be ""
//
//go:noinline
func DbInsert(when When, dbName string, tableName string, row Columns) BeforeAfter {
	return &beforeAfter{
		when: when,
		do: &dbInsert{
			dbName:    dbName,
			tableName: tableName,
			row:       row,
			frame:     framing.NewFrame(0),
		},
	}
}

// DbExec is a before/after operation to execute a statement on a database
//
// Note: when only one database is used by tests, the dbName can be ""
//
//go:noinline
func DbExec(when When, dbName string, query string, args ...any) BeforeAfter {
	return &beforeAfter{
		when: when,
		do: &dbExec{
			dbName: dbName,
			query:  query,
			args:   args,
			frame:  framing.NewFrame(0),
		},
	}
}

// DbClearTable is a before/after operation to clear a table in a database
//
// Note: when only one database is used by tests, the dbName can be ""
//
//go:noinline
func DbClearTable(when When, dbName string, tableName string) BeforeAfter {
	return &beforeAfter{
		when: when,
		do: &dbClearTable{
			dbName:    dbName,
			tableName: tableName,
			frame:     framing.NewFrame(0),
		},
	}
}

// MockServicesClearAll is a before/after operation to clear all mock services
//
//go:noinline
func MockServicesClearAll(when When) BeforeAfter {
	return &beforeAfter{
		when: when,
		do: &mockServicesClearAll{
			frame: framing.NewFrame(0),
		},
	}
}

// MockServiceClear is a before/after operation to clear a specific named mock service
//
//go:noinline
func MockServiceClear(when When, svcName string) BeforeAfter {
	return &beforeAfter{
		when: when,
		do: &mockServiceClear{
			name:  svcName,
			frame: framing.NewFrame(0),
		},
	}
}

// MockServiceCall is a before/after operation to set up a mock response on a specific named mock service
//
//go:noinline
func MockServiceCall(when When, svcName string, path string, method MethodName, responseStatus int, responseBody any, headers ...any) BeforeAfter {
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

// Wait is a before/after operation to wait a specified milliseconds
//
// Note: the wait time is not included in the coverage timings
//
//go:noinline
func Wait(when When, ms int) BeforeAfter {
	return &beforeAfter{
		when: when,
		do: &wait{
			ms:    ms,
			frame: framing.NewFrame(0),
		},
	}
}
