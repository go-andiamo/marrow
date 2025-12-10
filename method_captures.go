package marrow

import (
	"github.com/go-andiamo/marrow/framing"
	"strings"
	"time"
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
	// WaitFor waits for a specified condition to be met
	//
	// Notes:
	//   * the condition arg can be a bool value (or value that resolves to a bool) or an Expectation (e.g. ExpectEqual, ExpectNotEqual, etc.)
	//   * if the condition arg is an Expectation, and the expectation is unmet, this does not report a failure or unmet, instead the condition is re-evaluated until maxTime is exceeded
	//   * any condition that is not a bool or Expectation will cause an error during tests
	//
	// delays specifies the durations to wait on each poll cycle...
	//   * if not delays specified, there is no initial delay and polls occur every 250ms
	//   * if only one delay specified, there is no initial delay and polls occur at that specified duration
	//   * if more than one delay is specified, the initial delay is the first duration and subsequent poll delays are the remaining durations
	WaitFor(when When, condition any, maxTime time.Duration, delays ...time.Duration) Method_
	// Do adds before/after operations
	Do(ops ...BeforeAfter) Method_
	// Capture adds a before/after operations
	Capture(when When, ops ...Runnable) Method_
	// CaptureFunc adds the provided func(s) as a before/after operation
	CaptureFunc(when When, fns ...func(Context) error) Method_

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
func (m *method) WaitFor(when When, condition any, maxTime time.Duration, delays ...time.Duration) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &waitFor{
			condition: condition,
			maxTime:   maxTime,
			delays:    delays,
			frame:     framing.NewFrame(0),
		})
	} else {
		m.addPostCapture(&waitFor{
			condition: condition,
			maxTime:   maxTime,
			delays:    delays,
			frame:     framing.NewFrame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) Capture(when When, ops ...Runnable) Method_ {
	if when == Before {
		for _, op := range ops {
			if op == nil {
				continue
			}
			m.preCaptures = append(m.preCaptures, op)
		}
	} else {
		for _, op := range ops {
			if op == nil {
				continue
			}
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
func (m *method) CaptureFunc(when When, fns ...func(ctx Context) error) Method_ {
	if when == Before {
		for _, fn := range fns {
			if fn == nil {
				continue
			}
			m.preCaptures = append(m.preCaptures, &userDefinedCapture{
				fn:    fn,
				frame: framing.NewFrame(0),
			})
		}
	} else {
		for _, fn := range fns {
			if fn == nil {
				continue
			}
			m.addPostCapture(&userDefinedCapture{
				fn:    fn,
				frame: framing.NewFrame(0),
			})
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
