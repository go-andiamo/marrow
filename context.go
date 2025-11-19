package marrow

import (
	gctx "context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/mocks/service"
	"github.com/go-andiamo/marrow/testing"
	"github.com/go-andiamo/marrow/with"
	"maps"
	"net/http"
	"net/http/httptrace"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Context interface {
	// Host returns the currently tested API host
	Host() string
	// Vars returns the current Context variables
	//
	// the returned map is mutable but has no effect on the actual variables
	Vars() map[Var]any
	// Db returns a *sql.DB for the named database
	//
	// Note: when only one database is used by tests, the dbName can be ""
	Db(dbName string) *sql.DB
	// Ctx returns the go context.Context
	Ctx() gctx.Context
	// SetVar sets a variable in the context
	SetVar(Var, any)
	// ClearVars clears all variables in the context
	ClearVars()
	// CurrentEndpoint returns the current Endpoint being tested
	//
	// Note: may be nil if an endpoint not yet started
	CurrentEndpoint() Endpoint_
	// CurrentMethod returns the current Method being tested
	//
	// Note: may be nil if a method not yet started
	CurrentMethod() Method_
	// CurrentUrl returns the current URL for the current Endpoint
	CurrentUrl() string
	// CurrentRequest returns the current built request for a method
	//
	// Note: may be nil if the request has not yet been built
	CurrentRequest() *http.Request
	// CurrentResponse returns the current response for a method call
	//
	// Note: may be nil if the method has not yet been called
	CurrentResponse() *http.Response
	// CurrentBody returns the current unmarshalled response body for a method call
	//
	// Note: may be nil if the method has not yet been called or the response body was empty
	CurrentBody() any
	// DbInsert performs an insert into a database table
	//
	// Note: when only one database is used by tests, the dbName can be ""
	DbInsert(dbName string, tableName string, row Columns) error
	// DbExec executes a statement on a database
	//
	// Note: when only one database is used by tests, the dbName can be ""
	DbExec(dbName string, query string, args ...any) error
	// StoreCookie stores the cookie for later use
	StoreCookie(cookie *http.Cookie)
	// GetCookie returns a specific named cookie (or nil if that cookie has not been stored)
	GetCookie(name string) *http.Cookie
	// GetMockService returns a specific named mock service (or nil if that name is not registered)
	GetMockService(name string) service.MockedService
	// ClearMockServices clears all mock services
	ClearMockServices()
	// GetImage returns the named supporting image
	GetImage(name string) with.Image

	setCurrentEndpoint(Endpoint_)
	setCurrentMethod(Method_)
	setCurrentBody(any)
	doRequest(*http.Request) (*http.Response, bool)
	reportFailure(err error)
	reportUnmet(exp Expectation, err error)
	reportMet(exp Expectation)
	reportSkipped(exp Expectation)

	run(name string, r Runnable) bool
}

type context struct {
	coverage     coverage.Collector
	httpDo       common.HttpDo
	traceTimings bool
	host         string
	vars         map[Var]any
	dbs          namedDatabases
	images       map[string]with.Image
	testing      testing.Helper
	currTesting  []testing.Helper
	currEndpoint Endpoint_
	currMethod   Method_
	currRequest  *http.Request
	currResponse *http.Response
	currBody     any
	cookieJar    map[string]*http.Cookie
	mockServices map[string]service.MockedService
	failed       bool
}

func newContext() *context {
	return &context{
		coverage:     coverage.NewNullCoverage(),
		dbs:          make(namedDatabases),
		images:       make(map[string]with.Image),
		vars:         make(map[Var]any),
		cookieJar:    make(map[string]*http.Cookie),
		httpDo:       http.DefaultClient,
		mockServices: make(map[string]service.MockedService),
	}
}

var _ Context = (*context)(nil)

func (c *context) Host() string {
	return c.host
}

func (c *context) Vars() map[Var]any {
	return maps.Clone(c.vars)
}

func (c *context) Db(dbNamw string) *sql.DB {
	if tdb, ok := c.dbs[dbNamw]; ok {
		return tdb.db
	}
	return nil
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

func defaultStr(actual string, def string) string {
	if actual != "" {
		return actual
	}
	return def
}

func (c *context) DbInsert(dbName string, tableName string, row Columns) (err error) {
	tdb := c.dbs[dbName]
	if tdb == nil {
		return fmt.Errorf("db name %q not found", dbName)
	}
	db := tdb.db
	argMarkers := tdb.argMarkers
	args := make([]any, 0, len(row))
	markers := make([]string, 0, len(row))
	cols := make([]string, 0, len(row))
	i := argMarkers.Base
	addMarker := func(name string) {
		switch argMarkers.Style {
		case common.NumberedDbArgs:
			markers = append(markers, defaultStr(argMarkers.Prefix, "$")+strconv.Itoa(i))
			i++
		case common.NamedDbArgs:
			markers = append(markers, defaultStr(argMarkers.Prefix, ":")+name)
		default:
			markers = append(markers, defaultStr(argMarkers.Prefix, "?"))
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
			if to.Kind() == reflect.Map || to.Kind() == reflect.Slice || to.Kind() == reflect.Struct {
				var data []byte
				if data, err = json.Marshal(v); err == nil {
					addMarker(k)
					args = append(args, string(data))
				}
			} else {
				var av any
				if av, err = ResolveValue(v, c); err == nil {
					addMarker(k)
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
		_, err = db.Exec(query, args...)
	}
	return err
}

func (c *context) DbExec(dbName string, query string, args ...any) (err error) {
	tdb := c.dbs[dbName]
	if tdb == nil {
		return fmt.Errorf("db name %q not found", dbName)
	}
	db := tdb.db
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
		_, err = db.Exec(query, avArgs...)
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

func (c *context) GetMockService(name string) service.MockedService {
	return c.mockServices[name]
}

func (c *context) ClearMockServices() {
	for _, ms := range c.mockServices {
		ms.Clear()
	}
}

func (c *context) GetImage(name string) with.Image {
	return c.images[name]
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
	c.currRequest = nil
	if m != nil {
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
	var tt *coverage.TraceTiming
	useRequest := request
	if c.traceTimings {
		tt = &coverage.TraceTiming{}
		trace := &httptrace.ClientTrace{
			DNSStart:          func(_ httptrace.DNSStartInfo) { tt.DNSStart = time.Now() },
			DNSDone:           func(_ httptrace.DNSDoneInfo) { tt.DNSDone = time.Now() },
			ConnectStart:      func(_, _ string) { tt.ConnStart = time.Now() },
			ConnectDone:       func(_, _ string, _ error) { tt.ConnDone = time.Now() },
			TLSHandshakeStart: func() { tt.TLSStart = time.Now() },
			TLSHandshakeDone:  func(_ tls.ConnectionState, _ error) { tt.TLSDone = time.Now() },
			GotConn: func(info httptrace.GotConnInfo) {
				tt.ReusedConn = info.Reused
			},
			WroteRequest: func(_ httptrace.WroteRequestInfo) { tt.WroteReq = time.Now() },
			GotFirstResponseByte: func() {
				tt.FirstByte = time.Now()
				tt.TTFB = tt.FirstByte.Sub(tt.Start)
			},
		}
		useRequest = useRequest.WithContext(httptrace.WithClientTrace(useRequest.Context(), trace))
	}
	start := time.Now()
	if tt != nil {
		tt.Start = start
	}
	c.currResponse, err = c.httpDo.Do(useRequest)
	dur := time.Since(start)
	if err == nil {
		c.coverage.ReportTiming(c.currEndpoint, c.currMethod, c.currRequest, dur, tt)
		return c.currResponse, true
	}
	c.reportFailure(err)
	return nil, false
}

func (c *context) reportFailure(err error) {
	c.failed = true
	c.coverage.ReportFailure(c.currEndpoint, c.currMethod, c.currRequest, err)
	if currT := c.currentTest(); currT != nil {
		if eerr, ok := err.(Error); ok {
			currT.Log(eerr.TestFormat())
			currT.FailNow()
		} else {
			currT.Fatal(err)
		}
	}
}

func (c *context) reportUnmet(exp Expectation, err error) {
	c.coverage.ReportUnmet(c.currEndpoint, c.currMethod, c.currRequest, exp, err)
	if currT := c.currentTest(); currT != nil {
		if exp.IsRequired() {
			c.failed = true
			if umerr, ok := err.(UnmetError); ok {
				currT.Log(umerr.TestFormat())
				currT.FailNow()
			} else {
				currT.Fatal(err)
			}
		} else {
			if umerr, ok := err.(UnmetError); ok {
				currT.Log(umerr.TestFormat())
				currT.Fail()
			} else {
				currT.Error(err)
			}
		}
	} else if exp.IsRequired() {
		c.failed = true
	}
}

func (c *context) reportMet(exp Expectation) {
	c.coverage.ReportMet(c.currEndpoint, c.currMethod, c.currRequest, exp)
}

func (c *context) reportSkipped(exp Expectation) {
	c.coverage.ReportSkipped(c.currEndpoint, c.currMethod, c.currRequest, exp)
}

func (c *context) currentTest() testing.Helper {
	if c.testing == nil {
		return nil
	}
	result := c.testing
	if len(c.currTesting) > 0 {
		result = c.currTesting[len(c.currTesting)-1]
	}
	return result
}

func (c *context) run(name string, r Runnable) bool {
	c.failed = false
	if currT := c.currentTest(); currT != nil {
		currT.Run(name, func(t testing.Helper) {
			defer func() {
				c.currTesting = c.currTesting[:len(c.currTesting)-1]
			}()
			c.currTesting = append(c.currTesting, t)
			if err := r.Run(c); err != nil {
				c.reportFailure(err)
			}
		})
	} else {
		if err := r.Run(c); err != nil {
			c.reportFailure(err)
		}
	}
	return !c.failed
}
