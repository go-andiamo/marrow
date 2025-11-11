package marrow

import (
	"fmt"
	"github.com/go-andiamo/marrow/framing"
	"net/http"
	"os"
	"strings"
	"time"
)

type Capture interface {
	Name() string
	Runnable
}

type setVar struct {
	name  string
	value any
	frame *framing.Frame
}

var _ Capture = (*setVar)(nil)

func (c *setVar) Name() string {
	return c.name
}

func (c *setVar) Run(ctx Context) (err error) {
	var av any
	if av, err = ResolveValue(c.value, ctx); err == nil {
		ctx.SetVar(Var(c.name), av)
	}
	return wrapCaptureError(err, fmt.Sprintf("cannot set var %q", c.name), c, OperandValue{Original: c.value})
}

func (c *setVar) Frame() *framing.Frame {
	return c.frame
}

type clearVars struct {
	frame *framing.Frame
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

var _ Capture = (*clearVars)(nil)

type dbInsert struct {
	dbName    string
	tableName string
	row       Columns
	frame     *framing.Frame
}

var _ Capture = (*dbInsert)(nil)

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
