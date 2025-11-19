package marrow

import (
	"github.com/go-andiamo/marrow/framing"
	"strings"
)

type MethodCaptures interface {
	// SetVar sets a variable in the current Context
	SetVar(when When, name any, value any) Method_
	// ClearVars clears all variables in the current Context
	ClearVars(when When) Method_
	// DbInsert performs an insert into a database table
	//
	// Note: when only one database is used by tests, the dbName can be ""
	DbInsert(when When, dbName string, tableName string, row Columns) Method_
	// DbExec executes a statement on a database
	//
	// Note: when only one database is used by tests, the dbName can be ""
	DbExec(when When, dbName string, query string, args ...any) Method_
	// DbClearTables clears table(s) in a database
	//
	// Note: when only one database is used by tests, the dbName can be ""
	DbClearTables(when When, dbName string, tableNames ...string) Method_
	// Wait wait a specified milliseconds
	//
	// Note: the wait time is not included in the coverage timings
	Wait(when When, ms int) Method_
	// Capture adds a before/after operation
	Capture(op BeforeAfter) Method_
	// Do adds before/after operations
	Do(ops ...BeforeAfter) Method_
	// CaptureFunc adds the provided func as a before/after operation
	CaptureFunc(when When, fn func(Context) error) Method_
	// DoFunc adds the provided funcs as a before/after operation
	DoFunc(when When, fns ...func(Context) error) Method_

	// MockServicesClearAll clears all mock services
	MockServicesClearAll(when When) Method_
	// MockServiceClear clears a specific named mock service
	MockServiceClear(when When, svcName string) Method_
	// MockServiceCall sets up a mock response on a specific named mock service
	MockServiceCall(svcName string, path string, method MethodName, responseStatus int, responseBody any, headers ...any) Method_
}

//go:noinline
func (m *method) SetVar(when When, name any, value any) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &setVar{
			name:  name,
			value: value,
			frame: framing.NewFrame(0),
		})
	} else {
		m.addPostCapture(&setVar{
			name:  name,
			value: value,
			frame: framing.NewFrame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) ClearVars(when When) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &clearVars{
			frame: framing.NewFrame(0),
		})
	} else {
		m.addPostCapture(&clearVars{
			frame: framing.NewFrame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) DbInsert(when When, dbName string, tableName string, row Columns) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &dbInsert{
			dbName:    dbName,
			tableName: tableName,
			row:       row,
			frame:     framing.NewFrame(0),
		})
	} else {
		m.addPostCapture(&dbInsert{
			dbName:    dbName,
			tableName: tableName,
			row:       row,
			frame:     framing.NewFrame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) DbExec(when When, dbName string, query string, args ...any) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &dbExec{
			dbName: dbName,
			query:  query,
			args:   args,
			frame:  framing.NewFrame(0),
		})
	} else {
		m.addPostCapture(&dbExec{
			dbName: dbName,
			query:  query,
			args:   args,
			frame:  framing.NewFrame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) DbClearTables(when When, dbName string, tableNames ...string) Method_ {
	if when == Before {
		for _, tableName := range tableNames {
			m.preCaptures = append(m.preCaptures, &dbClearTable{
				dbName:    dbName,
				tableName: tableName,
				frame:     framing.NewFrame(0),
			})
		}
	} else {
		for _, tableName := range tableNames {
			m.addPostCapture(&dbClearTable{
				dbName:    dbName,
				tableName: tableName,
				frame:     framing.NewFrame(0),
			})
		}
	}
	return m
}

//go:noinline
func (m *method) Wait(when When, ms int) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &wait{
			ms:    ms,
			frame: framing.NewFrame(0),
		})
	} else {
		m.addPostCapture(&wait{
			ms:    ms,
			frame: framing.NewFrame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) Capture(op BeforeAfter) Method_ {
	if op != nil {
		if op.When() == Before {
			m.preCaptures = append(m.preCaptures, op)
		} else {
			m.addPostCapture(op)
		}
	}
	return m
}

//go:noinline
func (m *method) Do(ops ...BeforeAfter) Method_ {
	for _, op := range ops {
		if op != nil {
			if op.When() == Before {
				m.preCaptures = append(m.preCaptures, op)
			} else {
				m.addPostCapture(op)
			}
		}
	}
	return m
}

//go:noinline
func (m *method) CaptureFunc(when When, fn func(ctx Context) error) Method_ {
	if fn != nil {
		if when == Before {
			m.preCaptures = append(m.preCaptures, &userDefinedCapture{
				fn:    fn,
				frame: framing.NewFrame(0),
			})
		} else {
			m.addPostCapture(&userDefinedCapture{
				fn:    fn,
				frame: framing.NewFrame(0),
			})
		}
	}
	return m
}

//go:noinline
func (m *method) DoFunc(when When, fns ...func(ctx Context) error) Method_ {
	for _, fn := range fns {
		if fn != nil {
			if when == Before {
				m.preCaptures = append(m.preCaptures, &userDefinedCapture{
					fn:    fn,
					frame: framing.NewFrame(0),
				})
			} else {
				m.addPostCapture(&userDefinedCapture{
					fn:    fn,
					frame: framing.NewFrame(0),
				})
			}
		}
	}
	return m
}

//go:noinline
func (m *method) MockServicesClearAll(when When) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &mockServicesClearAll{
			frame: framing.NewFrame(0),
		})
	} else {
		m.addPostCapture(&mockServicesClearAll{
			frame: framing.NewFrame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) MockServiceClear(when When, svcName string) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &mockServiceClear{
			name:  svcName,
			frame: framing.NewFrame(0),
		})
	} else {
		m.addPostCapture(&mockServiceClear{
			name:  svcName,
			frame: framing.NewFrame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) MockServiceCall(svcName string, path string, method MethodName, responseStatus int, responseBody any, headers ...any) Method_ {
	m.preCaptures = append(m.preCaptures, &mockServiceCall{
		name:           svcName,
		path:           path,
		method:         strings.ToUpper(string(method)),
		responseStatus: responseStatus,
		responseBody:   responseBody,
		headers:        headers,
		frame:          framing.NewFrame(0),
	})
	return m
}
