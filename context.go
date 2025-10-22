package marrow

import (
	gctx "context"
	"database/sql"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type Context interface {
	Host() string
	Vars() map[Var]any
	Db() *sql.DB
	Ctx() gctx.Context
	SetVar(Var, any)
	ClearVars()
	CurrentEndpoint() Endpoint_
	CurrentMethod() Method_
	CurrentUrl() string
	CurrentRequest() *http.Request
	CurrentResponse() *http.Response
	CurrentBody() any
	DbInsert(tableName string, row Columns) error
	DbExec(query string, args ...any) error
	StoreCookie(cookie *http.Cookie)
	GetCookie(name string) *http.Cookie

	setCurrentEndpoint(Endpoint_)
	setCurrentMethod(Method_)
	setCurrentBody(any)
	doRequest(*http.Request) (*http.Response, bool)
	reportFailure(err error)
	reportUnmet(exp Expectation, err error)
	reportMet(exp Expectation)

	run(name string, r Runnable) bool
}

type DatabaseArgMarkers int

const (
	PositionalDbArgs DatabaseArgMarkers = iota
	NumberedDbArgs
)

type context struct {
	suite        Suite_
	coverage     *Coverage
	httpDo       HttpDo
	host         string
	vars         map[Var]any
	db           *sql.DB
	testing      *testing.T
	currTesting  []*testing.T
	dbArgMarkers DatabaseArgMarkers
	currEndpoint Endpoint_
	currMethod   Method_
	currRequest  *http.Request
	currResponse *http.Response
	currBody     any
	cookieJar    map[string]*http.Cookie
}

var _ Context = (*context)(nil)

func (c *context) Host() string {
	return c.host
}

func (c *context) Vars() map[Var]any {
	return c.vars
}

func (c *context) Db() *sql.DB {
	return c.db
}

func (c *context) Ctx() gctx.Context {
	return gctx.Background()
}

func (c *context) CurrentEndpoint() Endpoint_ {
	return c.currEndpoint
}

func (c *context) CurrentMethod() Method_ {
	return c.currMethod
}

func (c *context) CurrentUrl() string {
	if c.currEndpoint != nil {
		return c.currEndpoint.Url()
	}
	return ""
}

func (c *context) CurrentRequest() *http.Request {
	return c.currRequest
}

func (c *context) CurrentResponse() *http.Response {
	return c.currResponse
}

func (c *context) CurrentBody() any {
	return c.currBody
}

func (c *context) SetVar(name Var, value any) {
	c.vars[name] = value
}

func (c *context) ClearVars() {
	c.vars = make(map[Var]any)
}

type Columns map[string]any
type RawQuery string

func (c *context) DbInsert(tableName string, row Columns) (err error) {
	args := make([]any, 0, len(row))
	markers := make([]string, 0, len(row))
	cols := make([]string, 0, len(row))
	i := 0
	addMarker := func() {
		if c.dbArgMarkers == PositionalDbArgs {
			markers = append(markers, "?")
		} else {
			i++
			markers = append(markers, "$"+strconv.Itoa(i))
		}
	}
	for k, v := range row {
		cols = append(cols, k)
		switch vt := v.(type) {
		case RawQuery:
			var rq string
			if rq, err = resolveValueString(string(vt), c); err == nil {
				markers = append(markers, "("+rq+")")
			}
		default:
			to := reflect.TypeOf(v)
			if to.Kind() == reflect.Map || to.Kind() == reflect.Slice {
				var data []byte
				if data, err = json.Marshal(v); err == nil {
					addMarker()
					args = append(args, string(data))
				}
			} else {
				var av any
				if av, err = ResolveValue(v, c); err == nil {
					addMarker()
					args = append(args, av)
				}
			}
		}
		if err != nil {
			break
		}
	}
	if err == nil {
		query := "INSERT INTO " + tableName + " (" + strings.Join(cols, ",") + ") VALUES (" + strings.Join(markers, ",") + ")"
		_, err = c.db.Exec(query, args...)
	}
	return err
}

func (c *context) DbExec(query string, args ...any) (err error) {
	avArgs := make([]any, len(args))
	for i, v := range args {
		var av any
		if av, err = ResolveValue(v, c); err == nil {
			avArgs[i] = av
		} else {
			break
		}
	}
	if err == nil {
		_, err = c.db.Exec(query, avArgs...)
	}
	return err
}

func (c *context) StoreCookie(cookie *http.Cookie) {
	if cookie != nil {
		c.cookieJar[cookie.Name] = cookie
	}
}

func (c *context) GetCookie(name string) *http.Cookie {
	return c.cookieJar[name]
}

func (c *context) setCurrentEndpoint(e Endpoint_) {
	c.currEndpoint = e
	c.currMethod = nil
	c.currRequest = nil
	c.currResponse = nil
	c.currBody = nil
}

func (c *context) setCurrentMethod(m Method_) {
	c.currMethod = m
	if m != nil {
		c.currRequest = nil
		c.currResponse = nil
		c.currBody = nil
	}
}

func (c *context) setCurrentBody(body any) {
	c.currBody = body
}

func (c *context) doRequest(request *http.Request) (*http.Response, bool) {
	c.currRequest = request
	var err error
	if c.currResponse, err = c.httpDo.Do(request); err == nil {
		return c.currResponse, true
	}
	c.reportFailure(err)
	return nil, false
}

func (c *context) reportFailure(err error) {
	c.coverage.reportFailure(c.currEndpoint, c.currMethod, err)
	if len(c.currTesting) > 0 {
		currT := c.currTesting[len(c.currTesting)-1]
		if eerr, ok := err.(Error); ok {
			currT.Log(eerr.TestFormat())
			currT.FailNow()
		} else {
			currT.Fatal(err)
		}
	}
}

func (c *context) reportUnmet(exp Expectation, err error) {
	c.coverage.reportUnmet(c.currEndpoint, c.currMethod, exp, err)
	if len(c.currTesting) > 0 {
		currT := c.currTesting[len(c.currTesting)-1]
		if umerr, ok := err.(UnmetError); ok {
			currT.Log(umerr.TestFormat())
			currT.Fail()
		} else {
			currT.Error(err)
		}
	}
}

func (c *context) reportMet(exp Expectation) {
	c.coverage.reportMet(c.currEndpoint, c.currMethod, exp)
}

func (c *context) run(name string, r Runnable) (ok bool) {
	ok = true
	if c.testing != nil {
		currT := c.testing
		if len(c.currTesting) > 0 {
			currT = c.currTesting[len(c.currTesting)-1]
		}
		currT.Run(name, func(t *testing.T) {
			defer func() {
				c.currTesting = c.currTesting[:len(c.currTesting)-1]
			}()
			t.Helper()
			c.currTesting = append(c.currTesting, t)
			if err := r.Run(c); err != nil {
				ok = false
				c.reportFailure(err)
			}
		})
	} else {
		if err := r.Run(c); err != nil {
			ok = false
			c.reportFailure(err)
		}
	}
	return ok
}
