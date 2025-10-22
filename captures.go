package marrow

import (
	"fmt"
	"net/http"
)

type Capture interface {
	Name() string
	Runnable
}

type setVar struct {
	name  string
	value any
	frame *Frame
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

func (c *setVar) Frame() *Frame {
	return c.frame
}

type clearVars struct {
	frame *Frame
}

func (c *clearVars) Name() string {
	return "CLEAR VARS"
}

func (c *clearVars) Run(ctx Context) error {
	ctx.ClearVars()
	return nil
}

func (c *clearVars) Frame() *Frame {
	return c.frame
}

var _ Capture = (*clearVars)(nil)

type dbInsert struct {
	tableName string
	row       Columns
	frame     *Frame
}

var _ Capture = (*dbInsert)(nil)

func (c *dbInsert) Name() string {
	return "INSERT " + c.tableName
}

func (c *dbInsert) Run(ctx Context) error {
	return wrapCaptureError(ctx.DbInsert(c.tableName, c.row), "", c)
}

func (c *dbInsert) Frame() *Frame {
	return c.frame
}

type dbExec struct {
	query string
	args  []any
	frame *Frame
}

var _ Capture = (*dbExec)(nil)

func (c *dbExec) Name() string {
	return "EXEC " + c.query
}

func (c *dbExec) Run(ctx Context) error {
	return wrapCaptureError(ctx.DbExec(c.query, c.args...), "", c)
}

func (c *dbExec) Frame() *Frame {
	return c.frame
}

type dbClearTable struct {
	tableName string
	frame     *Frame
}

var _ Capture = (*dbClearTable)(nil)

func (c *dbClearTable) Name() string {
	return "DELETE FROM " + c.tableName
}

func (c *dbClearTable) Run(ctx Context) error {
	return wrapCaptureError(ctx.DbExec("DELETE FROM "+c.tableName), "", c)
}

func (c *dbClearTable) Frame() *Frame {
	return c.frame
}

type userDefinedCapture struct {
	name  string
	fn    func(ctx Context) error
	frame *Frame
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

func (c *userDefinedCapture) Frame() *Frame {
	return c.frame
}

type setCookie struct {
	cookie *http.Cookie
	frame  *Frame
}

var _ Capture = (*setCookie)(nil)

func (c *setCookie) Name() string {
	return "SET COOKIE " + c.cookie.Name
}

func (c *setCookie) Run(ctx Context) error {
	ctx.StoreCookie(c.cookie)
	return nil
}

func (c *setCookie) Frame() *Frame {
	return c.frame
}

type storeCookie struct {
	name  string
	frame *Frame
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

func (c *storeCookie) Frame() *Frame {
	return c.frame
}
