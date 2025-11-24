package marrow

import (
	"fmt"
	"github.com/go-andiamo/marrow/framing"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

type Capture interface {
	Name() string
	Runnable
}

type setVar struct {
	name  any
	value any
	frame *framing.Frame
}

var _ Capture = (*setVar)(nil)

// SetVar sets a variable in the current Context
//
//go:noinline
func SetVar(name any, value any) Capture {
	return &setVar{
		name:  name,
		value: value,
		frame: framing.NewFrame(0),
	}
}

func (c *setVar) Name() string {
	return c.nameString()
}

func (c *setVar) Run(ctx Context) (err error) {
	var av any
	if av, err = ResolveValue(c.value, ctx); err == nil {
		ctx.SetVar(Var(c.nameString()), av)
	}
	return wrapCaptureError(err, fmt.Sprintf("cannot set var %q", c.name), c, OperandValue{Original: c.value})
}

func (c *setVar) nameString() string {
	switch nt := c.name.(type) {
	case string:
		return nt
	case Var:
		return string(nt)
	default:
		return fmt.Sprintf("%v", c.name)
	}
}

func (c *setVar) Frame() *framing.Frame {
	return c.frame
}

type clearVars struct {
	frame *framing.Frame
}

var _ Capture = (*clearVars)(nil)

// ClearVars clears all variables in the current Context
//
//go:noinline
func ClearVars() Capture {
	return &clearVars{
		frame: framing.NewFrame(0),
	}
}

func (c *clearVars) Name() string {
	return "CLEAR VARS"
}

func (c *clearVars) Run(ctx Context) error {
	ctx.ClearVars()
	return nil
}

func (c *clearVars) Frame() *framing.Frame {
	return c.frame
}

type dbInsert struct {
	dbName    string
	tableName string
	row       Columns
	frame     *framing.Frame
}

var _ Capture = (*dbInsert)(nil)

// DbInsert is used to insert into a database table
//
// Note: when only one database is used by tests, the dbName can be ""
//
//go:noinline
func DbInsert(dbName string, tableName string, row Columns) Capture {
	return &dbInsert{
		dbName:    dbName,
		tableName: tableName,
		row:       row,
		frame:     framing.NewFrame(0),
	}
}

func (c *dbInsert) Name() string {
	return "INSERT " + c.tableName
}

func (c *dbInsert) Run(ctx Context) error {
	return wrapCaptureError(ctx.DbInsert(c.dbName, c.tableName, c.row), "", c)
}

func (c *dbInsert) Frame() *framing.Frame {
	return c.frame
}

type dbExec struct {
	dbName string
	query  string
	args   []any
	frame  *framing.Frame
}

var _ Capture = (*dbExec)(nil)

// DbExec is used to execute a statement on a database
//
// Note: when only one database is used by tests, the dbName can be ""
//
//go:noinline
func DbExec(dbName string, query string, args ...any) Capture {
	return &dbExec{
		dbName: dbName,
		query:  query,
		args:   args,
		frame:  framing.NewFrame(0),
	}
}

func (c *dbExec) Name() string {
	return "EXEC " + c.query
}

func (c *dbExec) Run(ctx Context) error {
	return wrapCaptureError(ctx.DbExec(c.dbName, c.query, c.args...), "", c)
}

func (c *dbExec) Frame() *framing.Frame {
	return c.frame
}

type dbClearTable struct {
	dbName    string
	tableName string
	frame     *framing.Frame
}

var _ Capture = (*dbClearTable)(nil)

// DbClearTable is used to clear a table in a database
//
// Note: when only one database is used by tests, the dbName can be ""
//
//go:noinline
func DbClearTable(dbName string, tableName string) Capture {
	return &dbClearTable{
		dbName:    dbName,
		tableName: tableName,
		frame:     framing.NewFrame(0),
	}
}

func (c *dbClearTable) Name() string {
	return "DELETE FROM " + c.tableName
}

func (c *dbClearTable) Run(ctx Context) error {
	return wrapCaptureError(ctx.DbExec(c.dbName, "DELETE FROM "+c.tableName), "", c)
}

func (c *dbClearTable) Frame() *framing.Frame {
	return c.frame
}

type userDefinedCapture struct {
	name  string
	fn    func(ctx Context) error
	frame *framing.Frame
}

var _ Capture = (*userDefinedCapture)(nil)

func (c *userDefinedCapture) Name() string {
	if c.name != "" {
		return c.name
	}
	return "(User Defined PreCapture)"
}

func (c *userDefinedCapture) Run(ctx Context) error {
	return wrapCaptureError(c.fn(ctx), "", c)
}

func (c *userDefinedCapture) Frame() *framing.Frame {
	return c.frame
}

type setCookie struct {
	cookie *http.Cookie
	frame  *framing.Frame
}

var _ Capture = (*setCookie)(nil)

// SetCookie sets a cookie in the current Context
//
//go:noinline
func SetCookie(cookie *http.Cookie) Capture {
	return &setCookie{
		cookie: cookie,
		frame:  framing.NewFrame(0),
	}
}

func (c *setCookie) Name() string {
	return "SET COOKIE " + c.cookie.Name
}

func (c *setCookie) Run(ctx Context) error {
	ctx.StoreCookie(c.cookie)
	return nil
}

func (c *setCookie) Frame() *framing.Frame {
	return c.frame
}

type storeCookie struct {
	name  string
	frame *framing.Frame
}

var _ Capture = (*storeCookie)(nil)

func (c *storeCookie) Name() string {
	return "STORE COOKIE " + c.name
}

func (c *storeCookie) Run(ctx Context) error {
	if response := ctx.CurrentResponse(); response == nil {
		return newCaptureError("response is nil", nil, c)
	} else {
		for _, cookie := range response.Cookies() {
			if cookie.Name == c.name {
				ctx.StoreCookie(cookie)
				return nil
			}
		}
	}
	return newCaptureError(fmt.Sprintf("no such cookie %q", c.name), nil, c)
}

func (c *storeCookie) Frame() *framing.Frame {
	return c.frame
}

type mockServicesClearAll struct {
	frame *framing.Frame
}

var _ Capture = (*mockServicesClearAll)(nil)

// MockServicesClearAll is used to clear all mock services
//
//go:noinline
func MockServicesClearAll() Capture {
	return &mockServicesClearAll{
		frame: framing.NewFrame(0),
	}
}

func (m *mockServicesClearAll) Name() string {
	return "CLEAR ALL MOCK SERVICES"
}

func (m *mockServicesClearAll) Run(ctx Context) error {
	ctx.ClearMockServices()
	return nil
}

func (m *mockServicesClearAll) Frame() *framing.Frame {
	return m.frame
}

type mockServiceClear struct {
	name  string
	frame *framing.Frame
}

var _ Capture = (*mockServiceClear)(nil)

// MockServiceClear is used to clear a specific named mock service
//
//go:noinline
func MockServiceClear(svcName string) Capture {
	return &mockServiceClear{
		name:  svcName,
		frame: framing.NewFrame(0),
	}
}

func (m *mockServiceClear) Name() string {
	return "CLEAR MOCK SERVICE [" + m.name + "]"
}

func (m *mockServiceClear) Run(ctx Context) error {
	if ms := ctx.GetMockService(m.name); ms != nil {
		ms.Clear()
		return nil
	}
	return newCaptureError(fmt.Sprintf("unknown mock service %q", m.name), nil, m)
}

func (m *mockServiceClear) Frame() *framing.Frame {
	return m.frame
}

type mockServiceCall struct {
	name           string
	path           string
	method         string
	responseStatus int
	responseBody   any
	headers        []any
	frame          *framing.Frame
}

var _ Capture = (*mockServiceCall)(nil)

// MockServiceCall is used to set up a mock response on a specific named mock service
//
//go:noinline
func MockServiceCall(svcName string, path string, method MethodName, responseStatus int, responseBody any, headers ...any) Capture {
	return &mockServiceCall{
		name:           svcName,
		path:           path,
		method:         strings.ToUpper(string(method)),
		responseStatus: responseStatus,
		responseBody:   responseBody,
		headers:        headers,
		frame:          framing.NewFrame(0),
	}
}

func (m *mockServiceCall) Name() string {
	return "MOCK SERVICE CALL [" + m.name + "]: " + m.method + " " + m.path
}

func (m *mockServiceCall) Run(ctx Context) (err error) {
	if ms := ctx.GetMockService(m.name); ms != nil {
		var actualPath string
		if actualPath, err = resolveValueString(m.path, ctx); err == nil {
			var actualBody any
			if actualBody, err = ResolveValue(m.responseBody, ctx); err == nil {
				actualHdrs := make([]string, 0, len(m.headers))
				for h := 0; h < len(m.headers) && err == nil; h += 2 {
					actualHdrs = append(actualHdrs, fmt.Sprintf("%v", m.headers[h]))
					var hv any
					if hv, err = ResolveValue(m.headers[h+1], ctx); err == nil {
						actualHdrs = append(actualHdrs, fmt.Sprintf("%v", hv))
					}
				}
				if err == nil {
					ms.MockCall(actualPath, m.method, m.responseStatus, actualBody, actualHdrs...)
				}
			}
		}
		err = wrapCaptureError(err, "", m)
		return err
	}
	return newCaptureError(fmt.Sprintf("unknown mock service %q", m.name), nil, m)
}

func (m *mockServiceCall) Frame() *framing.Frame {
	return m.frame
}

type wait struct {
	ms    int
	frame *framing.Frame
}

var _ Capture = (*wait)(nil)

// Wait is used to wait a specified milliseconds
//
// Note: the wait time is not included in the coverage timings
//
//go:noinline
func Wait(ms int) Capture {
	return &wait{
		ms:    ms,
		frame: framing.NewFrame(0),
	}
}

func (w *wait) Name() string {
	return fmt.Sprintf("WAIT %s", time.Duration(w.ms)*time.Millisecond)
}

func (w *wait) Run(ctx Context) error {
	time.Sleep(time.Duration(w.ms) * time.Millisecond)
	return nil
}

func (w *wait) Frame() *framing.Frame {
	return w.frame
}

type setEnv struct {
	name  string
	value any
	frame *framing.Frame
}

var _ Capture = (*setEnv)(nil)

// SetEnv is used to set an environment variable
//
//go:noinline
func SetEnv(name string, value any) Capture {
	return &setEnv{
		name:  name,
		value: value,
		frame: framing.NewFrame(0),
	}
}

func (s *setEnv) Name() string {
	return fmt.Sprintf("SET ENV: %q", s.name)
}

func (s *setEnv) Run(ctx Context) (err error) {
	var av any
	if av, err = ResolveValue(s.value, ctx); err == nil {
		sv := ""
		switch avt := av.(type) {
		case string:
			sv = avt
		default:
			sv = fmt.Sprintf("%v", av)
		}
		err = os.Setenv(s.name, sv)
	}
	return err
}

func (s *setEnv) Frame() *framing.Frame {
	return s.frame
}

type unSetEnv struct {
	names []string
	frame *framing.Frame
}

var _ Capture = (*unSetEnv)(nil)

// UnSetEnv is used to unset an environment variable
//
//go:noinline
func UnSetEnv(names ...string) Capture {
	return &unSetEnv{
		names: names,
		frame: framing.NewFrame(0),
	}
}

func (u *unSetEnv) Name() string {
	return `UNSET ENV: "` + strings.Join(u.names, `", "`) + `"`
}

func (u *unSetEnv) Run(ctx Context) (err error) {
	for i := 0; i < len(u.names) && err == nil; i++ {
		err = os.Unsetenv(u.names[i])
	}
	return err
}

func (u *unSetEnv) Frame() *framing.Frame {
	return u.frame
}

type conditional struct {
	condition any
	not       bool
	ops       []Runnable
	frame     *framing.Frame
}

var _ Capture = (*conditional)(nil)

// If runs the operations when the condition arg is met
//
// Notes:
//   - the condition arg can be a bool value (or value that resolves to a bool) or an Expectation (e.g. ExpectEqual, ExpectNotEqual, etc.)
//   - if the condition arg is an Expectation, and the expectation is unmet, this does not report a failure or unmet, instead the operations are just not performed
//   - any condition that is not a bool or Expectation will cause an error during tests
//   - the operations arg can be anything Runnable - any of them that are an Expectation, is run as an expectation (and treated as required) and any unmet or failure errors will be reported
//
//go:noinline
func If(condition any, ops ...Runnable) Capture {
	return &conditional{
		condition: condition,
		ops:       ops,
		frame:     framing.NewFrame(0),
	}
}

// IfNot runs the operations when the condition arg is not met
//
// Notes:
//   - the condition arg can be a bool value (or value that resolves to a bool) or an Expectation (e.g. ExpectEqual, ExpectNotEqual, etc.)
//   - if the condition arg is an Expectation, and the expectation is unmet, this does not report a failure or unmet, instead the operations are just not performed
//   - any condition that is not a bool or Expectation will cause an error during tests
//   - the operations arg can be anything Runnable - any of them that are an Expectation, is run as an expectation (and treated as required) and any unmet or failure errors will be reported
//
//go:noinline
func IfNot(condition any, ops ...Runnable) Capture {
	return &conditional{
		condition: condition,
		not:       true,
		ops:       ops,
		frame:     framing.NewFrame(0),
	}
}

func (c *conditional) Name() string {
	return "CONDITIONAL"
}

func (c *conditional) Run(ctx Context) (err error) {
	do := false
	if exp, ok := c.condition.(Expectation); ok {
		var unmet error
		if c.not {
			if unmet, err = exp.Met(ctx); unmet != nil && err == nil {
				do = true
			}
		} else if unmet, err = exp.Met(ctx); unmet == nil && err == nil {
			do = true
		}
	} else {
		var av any
		if av, err = ResolveValue(c.condition, ctx); err == nil {
			if b, ok := av.(bool); ok {
				do = b
				if c.not {
					do = !do
				}
			} else {
				err = fmt.Errorf("invalid condition type: %T", av)
			}
		}
	}
	if do {
		for i, o := range c.ops {
			if o == nil {
				continue
			}
			pass := false
			if expOp, ok := c.ops[i].(Expectation); ok {
				var unmet error
				if unmet, err = expOp.Met(ctx); unmet == nil && err == nil {
					ctx.reportMet(expOp)
					pass = true
				} else if err != nil {
					ctx.reportFailure(err)
				} else {
					ctx.reportUnmet(expOp, unmet)
				}
			} else if err = o.Run(ctx); err == nil {
				pass = true
			}
			if !pass {
				for j := i + 1; j < len(c.ops); j++ {
					if expOp, ok := c.ops[j].(Expectation); ok {
						ctx.reportSkipped(expOp)
					}
				}
				break
			}
		}
	}
	return err
}

func (c *conditional) Frame() *framing.Frame {
	return c.frame
}

type forEach struct {
	value   any
	varName any
	ops     []Runnable
	frame   *framing.Frame
}

var _ Capture = (*forEach)(nil)

// ForEach iterates over the value (or resolved value)
//
// if the value (or resolved value) is not a slice - an error occurs
//
// on each iteration a var is set - use the iterVar arg to set the name of this variable (if the iterVar arg is nil - a var name of "." is used)
//
//go:noinline
func ForEach(value any, iterVar any, ops ...Runnable) Capture {
	return &forEach{
		value:   value,
		varName: iterVar,
		ops:     ops,
		frame:   framing.NewFrame(0),
	}
}

func (c *forEach) Name() string {
	return fmt.Sprintf("ForEach(%v)", c.value)
}

func (c *forEach) Run(ctx Context) (err error) {
	var av any
	if av, err = ResolveValue(c.value, ctx); err == nil {
		var items []any
		switch avt := av.(type) {
		case []any:
			items = avt
		default:
			if av != nil {
				to := reflect.ValueOf(av)
				if to.Kind() == reflect.Slice {
					l := to.Len()
					items = make([]any, l)
					for i := 0; i < l; i++ {
						items[i] = to.Index(i).Interface()
					}
				} else {
					return fmt.Errorf("invalid ForEach value type: %T", av)
				}
			}
		}
		varName := c.useVarName()
		for _, item := range items {
			ctx.SetVar(varName, item)
			for i, o := range c.ops {
				if o == nil {
					continue
				}
				pass := false
				if expOp, ok := c.ops[i].(Expectation); ok {
					var unmet error
					if unmet, err = expOp.Met(ctx); unmet == nil && err == nil {
						ctx.reportMet(expOp)
						pass = true
					} else if err != nil {
						ctx.reportFailure(err)
					} else {
						ctx.reportUnmet(expOp, unmet)
					}
				} else if err = o.Run(ctx); err == nil {
					pass = true
				}
				if !pass {
					for j := i + 1; j < len(c.ops); j++ {
						if expOp, ok := c.ops[j].(Expectation); ok {
							ctx.reportSkipped(expOp)
						}
					}
					break
				}
			}
		}
	}
	return err
}

func (c *forEach) Frame() *framing.Frame {
	return c.frame
}

func (c *forEach) useVarName() Var {
	switch nt := c.varName.(type) {
	case nil:
		return "."
	case string:
		return Var(nt)
	case Var:
		return nt
	default:
		return Var(fmt.Sprintf("%v", c.varName))
	}
}
