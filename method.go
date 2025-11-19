package marrow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-andiamo/urit"
	"io"
	"maps"
	"net/http"
	"strings"
)

type methodExpectations interface {
	// Expect adds a new expectation to the Method_
	Expect(exp Expectation) Method_

	// AssertOK asserts the response status code is 200 "OK"
	AssertOK() Method_
	// AssertCreated asserts the response status code is 201 "Created"
	AssertCreated() Method_
	// AssertAccepted asserts the response status code is 202 "Accepted"
	AssertAccepted() Method_
	// AssertNoContent asserts the response status code is 204 "No Content"
	AssertNoContent() Method_
	// AssertBadRequest asserts the response status code is 400 "Bad Request"
	AssertBadRequest() Method_
	// AssertUnauthorized asserts the response status code is 401 "Unauthorized"
	AssertUnauthorized() Method_
	// AssertForbidden asserts the response status code is 403 "Forbidden"
	AssertForbidden() Method_
	// AssertNotFound asserts the response status code is 404 "Not Found"
	AssertNotFound() Method_
	// AssertConflict asserts the response status code is 409 "Conflict"
	AssertConflict() Method_
	// AssertGone asserts the response status code is 410 "Gone"
	AssertGone() Method_
	// AssertUnprocessableEntity asserts the response status code is 422 "Unprocessable Entity"
	AssertUnprocessableEntity() Method_
	// AssertStatus asserts the response status code is the status supplied
	AssertStatus(status any) Method_
	// AssertFunc asserts that the supplied func is met
	AssertFunc(fn func(Context) (unmet error, err error)) Method_
	// AssertEqual asserts that the supplied values are equal
	//
	// values can be any of:
	//  * primitive type of string, bool, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	AssertEqual(v1, v2 any) Method_
	// AssertNotEqual asserts that the supplied values are not equal
	//
	// values can be any of:
	//  * primitive type of string, bool, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	AssertNotEqual(v1, v2 any) Method_
	// AssertLessThan asserts that v1 is less than v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	AssertLessThan(v1, v2 any) Method_
	// AssertLessThanOrEqual asserts that v1 is less than or equal to v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	AssertLessThanOrEqual(v1, v2 any) Method_
	// AssertGreaterThan asserts that v1 is greater than v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	AssertGreaterThan(v1, v2 any) Method_
	// AssertGreaterThanOrEqual asserts that v1 is greater than or equal to v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	AssertGreaterThanOrEqual(v1, v2 any) Method_
	// AssertNotLessThan asserts that v1 is not less than v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	AssertNotLessThan(v1, v2 any) Method_
	// AssertNotGreaterThan asserts that v1 is not greater than v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	AssertNotGreaterThan(v1, v2 any) Method_
	// AssertMatch asserts that the value matches the supplied regex
	//
	// when attempting to match against the regex, the value (or resolved value) is "stringified"
	//
	// values can be any of:
	//  * primitive type of string, bool, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	AssertMatch(value any, regex string) Method_
	// AssertContains asserts that the value contains a substring
	//
	// when attempting to check contains, the value (or resolved value) is "stringified"
	AssertContains(value any, s string) Method_
	// AssertType asserts that the value (or resolved value) is of the supplied type
	AssertType(value any, typ Type_) Method_
	// AssertNil asserts that the value (or resolved value) is nil
	AssertNil(value any) Method_
	// AssertNotNil asserts that the value (or resolved value) is not nil
	AssertNotNil(value any) Method_
	// AssertLen asserts that the value (or resolved value) has the supplied length
	//
	// the value (or resolved value) must be a string, map or slice
	AssertLen(value any, length int) Method_
	// AssertHasProperties asserts that the value (or resolved value) has the supplied properties
	//
	// the value (or resolved value) must be a map
	AssertHasProperties(value any, propertyNames ...string) Method_
	// AssertOnlyHasProperties asserts that the value (or resolved value) only has the supplied properties
	//
	// the value (or resolved value) must be a map
	AssertOnlyHasProperties(value any, propertyNames ...string) Method_

	// RequireOK requires the response status code is 200 "OK"
	RequireOK() Method_
	// RequireCreated requires the response status code is 201 "Created"
	RequireCreated() Method_
	// RequireAccepted requires the response status code is 202 "Accepted"
	RequireAccepted() Method_
	// RequireNoContent requires the response status code is 204 "No Content"
	RequireNoContent() Method_
	// RequireBadRequest requires the response status code is 400 "Bad Request"
	RequireBadRequest() Method_
	// RequireUnauthorized requires the response status code is 401 "Unauthorized"
	RequireUnauthorized() Method_
	// RequireForbidden requires the response status code is 403 "Forbidden"
	RequireForbidden() Method_
	// RequireNotFound requires the response status code is 404 "Not Found"
	RequireNotFound() Method_
	// RequireConflict requires the response status code is 409 "Conflict"
	RequireConflict() Method_
	// RequireGone requires the response status code is 410 "Gone"
	RequireGone() Method_
	// RequireUnprocessableEntity requires the response status code is 422 "Unprocessable Entity"
	RequireUnprocessableEntity() Method_
	// RequireStatus requires the response status code is the status supplied
	RequireStatus(status any) Method_
	// RequireFunc requires that the supplied func is met
	RequireFunc(fn func(Context) (unmet error, err error)) Method_
	// RequireEqual requires that the supplied values are equal
	//
	// values can be any of:
	//  * primitive type of string, bool, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	RequireEqual(v1, v2 any) Method_
	// RequireNotEqual requires that the supplied values are not equal
	//
	// values can be any of:
	//  * primitive type of string, bool, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	RequireNotEqual(v1, v2 any) Method_
	// RequireLessThan requires that v1 is less than v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	RequireLessThan(v1, v2 any) Method_
	// RequireLessThanOrEqual requires that v1 is less than or equal to v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	RequireLessThanOrEqual(v1, v2 any) Method_
	// RequireGreaterThan requires that v1 is greater than v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	RequireGreaterThan(v1, v2 any) Method_
	// RequireGreaterThanOrEqual requires that v1 is greater than or equal to v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	RequireGreaterThanOrEqual(v1, v2 any) Method_
	// RequireNotLessThan requires that v1 is not less than v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	RequireNotLessThan(v1, v2 any) Method_
	// RequireNotGreaterThan requires that v1 is not greater than v2
	//
	// values can be any of:
	//  * primitive type of string, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	RequireNotGreaterThan(v1, v2 any) Method_
	// RequireMatch requires that the value matches the supplied regex
	//
	// when attempting to match against the regex, the value (or resolved value) is "stringified"
	//
	// values can be any of:
	//  * primitive type of string, bool, int, int64, float64
	//  * decimal.Decimal
	//  * or anything that is resolvable...
	//
	// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
	// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
	RequireMatch(value any, regex string) Method_
	// RequireContains requires that the value contains a substring
	//
	// when attempting to check contains, the value (or resolved value) is "stringified"
	RequireContains(value any, s string) Method_
	// RequireType requires that the value (or resolved value) is of the supplied type
	RequireType(value any, typ Type_) Method_
	// RequireNil requires that the value (or resolved value) is nil
	RequireNil(value any) Method_
	// RequireNotNil requires that the value (or resolved value) is not nil
	RequireNotNil(value any) Method_
	// RequireLen requires that the value (or resolved value) has the supplied length
	//
	// the value (or resolved value) must be a string, map or slice
	RequireLen(value any, length int) Method_
	// RequireHasProperties requires that the value (or resolved value) has the supplied properties
	//
	// the value (or resolved value) must be a map
	RequireHasProperties(value any, propertyNames ...string) Method_
	// RequireOnlyHasProperties requires that the value (or resolved value) only has the supplied properties
	//
	// the value (or resolved value) must be a map
	RequireOnlyHasProperties(value any, propertyNames ...string) Method_

	// FailFast instructs the method to fail on unmet assertions
	//
	// i.e. treat all `Assert...()` as `Require...()`
	FailFast() Method_
}

type methodConditional interface {
	// If runs the operations when the condition arg is met
	//
	// Notes:
	//   * the condition arg can be a bool value (or value that resolves to a bool) or an Expectation (e.g. ExpectEqual, ExpectNotEqual, etc.)
	//   * if the condition arg is an Expectation, and the expectation is unmet, this does not report a failure or unmet, instead the operations are just not performed
	//   * any condition that is not a bool or Expectation will cause an error during tests
	//   * the operations arg can be anything Runnable - any of them that are an Expectation, is run as an expectation (and treated as required) and any unmet or failure errors will be reported
	If(when When, condition any, operations ...Runnable) Method_
}

type methodMockService interface {
	// MockServicesClearAll clears all mock services
	MockServicesClearAll(when When) Method_
	// MockServiceClear clears a specific named mock service
	MockServiceClear(when When, svcName string) Method_
	// MockServiceCall sets up a mock response on a specific named mock service
	MockServiceCall(svcName string, path string, method MethodName, responseStatus int, responseBody any, headers ...any) Method_
	// AssertMockServiceCalled asserts that a specific mock service endpoint+method was called
	AssertMockServiceCalled(svcName string, path string, method MethodName) Method_
	// RequireMockServiceCalled requires that a specific mock service endpoint+method was called
	RequireMockServiceCalled(svcName string, path string, method MethodName) Method_
}

// Method_ is the interface implemented by an instantiated Method (see Method(), Get(), Post(), etc.)
type Method_ interface {
	common.Method

	// QueryParam sets a http query param to value(s) for the method call
	QueryParam(name string, values ...any) Method_
	// PathParam sets a URL path param for the method call
	//
	// path params are specified in the order in which they appear in the url template
	PathParam(value any) Method_
	// RequestHeader sets a http header for the method call
	RequestHeader(name string, value any) Method_
	// RequestBody sets the http request body for the method call
	RequestBody(value any) Method_
	// UseCookie sets a named cookie to use on the method call
	//
	// the named cookie has to have been previously stored in the Context
	UseCookie(name string) Method_
	// SetCookie sets a cookie to be used for the method call
	SetCookie(cookie *http.Cookie) Method_
	// StoreCookie stores a named cookie from the response in the Context
	StoreCookie(name string) Method_
	// Authorize provides a function that can be used to authorize a http request
	//
	// the func is passed the current Context, and can obtain the built request using Context.CurrentRequest and manipulate it
	//
	// Notes:
	//  * multiple authorize functions can be added.
	//  * authorize functions are called after pre-captures (i.e. Before's) and after the request has been built
	Authorize(func(ctx Context) error) Method_

	methodExpectations
	methodMockService
	methodConditional

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
	// CaptureFunc adds the provided func as a before/after operation
	CaptureFunc(when When, fn func(Context) error) Method_

	// RequestMarshal provides an override function to build (marshal) the request body
	//
	// only one func is used, so if this is called multiple times - last one wins
	RequestMarshal(fn func(ctx Context, body any) ([]byte, error)) Method_
	// ResponseUnmarshal provides an override function to unmarshal the response body
	//
	// only one func is used, so if this is called multiple times - last one wins
	ResponseUnmarshal(fn func(response *http.Response) (any, error)) Method_
	Runnable
	fmt.Stringer
}

// Method instantiates a new method test
//
// verb arg is the http method, e.g. "GET", "PUT", "POST" etc.
//
// desc arg is the description of the method test
//
// ops args are any before/after operations to be run as part of the method test
//
//go:noinline
func Method(verb MethodName, desc string, ops ...BeforeAfter) Method_ {
	return newMethod(verb, desc, ops...)
}

// Get instantiates a new GET method test
//
// synonymous with calling Method(GET, ...)
//
//go:noinline
func Get(desc string, ops ...BeforeAfter) Method_ {
	return newMethod(GET, desc, ops...)
}

// Head instantiates a new HEAD method test
//
// synonymous with calling Method(HEAD, ...)
//
//go:noinline
func Head(desc string, ops ...BeforeAfter) Method_ {
	return newMethod(HEAD, desc, ops...)
}

// Post instantiates a new POST method test
//
// synonymous with calling Method(POST, ...)
//
//go:noinline
func Post(desc string, ops ...BeforeAfter) Method_ {
	return newMethod(POST, desc, ops...)
}

// Put instantiates a new PUT method test
//
// synonymous with calling Method(PUT, ...)
//
//go:noinline
func Put(desc string, ops ...BeforeAfter) Method_ {
	return newMethod(PUT, desc, ops...)
}

// Patch instantiates a new PATCH method test
//
// synonymous with calling Method(PATCH, ...)
//
//go:noinline
func Patch(desc string, ops ...BeforeAfter) Method_ {
	return newMethod(PATCH, desc, ops...)
}

// Delete instantiates a new DELETE method test
//
// synonymous with calling Method(DELETE, ...)
//
//go:noinline
func Delete(desc string, ops ...BeforeAfter) Method_ {
	return newMethod(DELETE, desc, ops...)
}

//go:noinline
func newMethod(m MethodName, desc string, ops ...BeforeAfter) Method_ {
	result := &method{
		desc:        desc,
		frame:       framing.NewFrame(1),
		method:      m.Normalize(),
		queryParams: queryParams{},
		pathParams:  pathParams{},
		headers:     make(map[string]any),
		useCookies:  make(map[string]struct{}),
	}
	for _, op := range ops {
		if op != nil {
			if op.When() == Before {
				result.preCaptures = append(result.preCaptures, op)
			} else {
				result.addPostCapture(op)
			}
		}
	}
	return result
}

type postOp struct {
	isExpectation bool
	index         int
}

type method struct {
	desc              string
	frame             *framing.Frame
	method            MethodName
	pathParams        pathParams
	queryParams       queryParams
	headers           map[string]any
	body              any
	preCaptures       []Runnable
	authFns           []Runnable
	postOps           []postOp
	postCaptures      []Runnable
	expectations      []Expectation
	failFast          bool
	useCookies        map[string]struct{}
	requestMarshal    func(ctx Context, body any) ([]byte, error)
	responseUnmarshal func(response *http.Response) (any, error)
}

func (m *method) addPostCapture(c Runnable) {
	m.postOps = append(m.postOps, postOp{
		isExpectation: false,
		index:         len(m.postCaptures),
	})
	m.postCaptures = append(m.postCaptures, c)
}

func (m *method) addPostExpectation(exp Expectation) {
	m.postOps = append(m.postOps, postOp{
		isExpectation: true,
		index:         len(m.expectations),
	})
	m.expectations = append(m.expectations, exp)
}

func (m *method) MethodName() string {
	return string(m.method)
}

func (m *method) Description() string {
	return m.desc
}

func (m *method) Frame() *framing.Frame {
	return m.frame
}

func (m *method) FailFast() Method_ {
	m.failFast = true
	return m
}

//go:noinline
func (m *method) Authorize(fn func(ctx Context) error) Method_ {
	if fn != nil {
		m.authFns = append(m.authFns, &userDefinedCapture{
			name:  "Authorize",
			fn:    fn,
			frame: framing.NewFrame(0),
		})
	}
	return m
}

func (m *method) QueryParam(name string, values ...any) Method_ {
	m.queryParams[name] = append(m.queryParams[name], values...)
	return m
}

func (m *method) PathParam(value any) Method_ {
	m.pathParams = append(m.pathParams, value)
	return m
}

func (m *method) RequestHeader(name string, value any) Method_ {
	m.headers[name] = value
	return m
}

func (m *method) RequestBody(value any) Method_ {
	m.body = value
	return m
}

func (m *method) UseCookie(name string) Method_ {
	m.useCookies[name] = struct{}{}
	return m
}

//go:noinline
func (m *method) SetCookie(cookie *http.Cookie) Method_ {
	if cookie != nil {
		m.preCaptures = append(m.preCaptures, &setCookie{
			cookie: cookie,
			frame:  framing.NewFrame(0),
		})
		m.useCookies[cookie.Name] = struct{}{}
	}
	return m
}

//go:noinline
func (m *method) StoreCookie(name string) Method_ {
	m.addPostCapture(&storeCookie{
		name:  name,
		frame: framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) AssertOK() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect OK",
		expect: http.StatusOK,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireOK() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect OK",
		expect:            http.StatusOK,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertCreated() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect Created",
		expect: http.StatusCreated,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireCreated() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect Created",
		expect:            http.StatusCreated,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertAccepted() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect Accepted",
		expect: http.StatusAccepted,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireAccepted() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect Accepted",
		expect:            http.StatusAccepted,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertNoContent() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect No Content",
		expect: http.StatusNoContent,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireNoContent() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect No Content",
		expect:            http.StatusNoContent,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertBadRequest() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect Bad Request",
		expect: http.StatusBadRequest,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireBadRequest() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect Bad Request",
		expect:            http.StatusBadRequest,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertUnauthorized() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect Unauthorized",
		expect: http.StatusUnauthorized,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireUnauthorized() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect Unauthorized",
		expect:            http.StatusUnauthorized,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertForbidden() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect Forbidden",
		expect: http.StatusForbidden,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireForbidden() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect Forbidden",
		expect:            http.StatusForbidden,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertNotFound() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect Not Found",
		expect: http.StatusNotFound,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireNotFound() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect Not Found",
		expect:            http.StatusNotFound,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertConflict() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect Conflict",
		expect: http.StatusConflict,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireConflict() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect Conflict",
		expect:            http.StatusConflict,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertGone() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect Gone",
		expect: http.StatusGone,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireGone() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect Gone",
		expect:            http.StatusGone,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertUnprocessableEntity() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect Unprocessable Entity",
		expect: http.StatusUnprocessableEntity,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireUnprocessableEntity() Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect Unprocessable Entity",
		expect:            http.StatusUnprocessableEntity,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertStatus(status any) Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:   "Expect Status Code",
		expect: status,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireStatus(status any) Method_ {
	m.addPostExpectation(&expectStatusCode{
		name:              "Expect Status Code",
		expect:            status,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
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
func (m *method) Expect(exp Expectation) Method_ {
	m.addPostExpectation(exp)
	return m
}

//go:noinline
func (m *method) AssertFunc(fn func(Context) (unmet error, err error)) Method_ {
	if fn != nil {
		m.addPostExpectation(&expectation{
			fn:    fn,
			frame: framing.NewFrame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) RequireFunc(fn func(Context) (unmet error, err error)) Method_ {
	if fn != nil {
		m.addPostExpectation(&expectation{
			fn:                fn,
			frame:             framing.NewFrame(0),
			commonExpectation: commonExpectation{required: true},
		})
	}
	return m
}

//go:noinline
func (m *method) AssertEqual(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectEqual", v1, v2, compEqual, false, false))
	return m
}

//go:noinline
func (m *method) RequireEqual(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectEqual", v1, v2, compEqual, false, true))
	return m
}

//go:noinline
func (m *method) AssertNotEqual(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectNotEqual", v1, v2, compEqual, true, false))
	return m
}

//go:noinline
func (m *method) RequireNotEqual(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectNotEqual", v1, v2, compEqual, true, true))
	return m
}

//go:noinline
func (m *method) AssertLessThan(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectLessThan", v1, v2, compLessThan, false, false))
	return m
}

//go:noinline
func (m *method) RequireLessThan(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectLessThan", v1, v2, compLessThan, false, true))
	return m
}

//go:noinline
func (m *method) AssertLessThanOrEqual(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectLessThanOrEqual", v1, v2, compLessOrEqualThan, false, false))
	return m
}

//go:noinline
func (m *method) RequireLessThanOrEqual(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectLessThanOrEqual", v1, v2, compLessOrEqualThan, false, true))
	return m
}

//go:noinline
func (m *method) AssertGreaterThan(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectGreaterThan", v1, v2, compGreaterThan, false, false))
	return m
}

//go:noinline
func (m *method) RequireGreaterThan(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectGreaterThan", v1, v2, compGreaterThan, false, true))
	return m
}

//go:noinline
func (m *method) AssertGreaterThanOrEqual(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectGreaterThanOrEqual", v1, v2, compGreaterOrEqualThan, false, false))
	return m
}

//go:noinline
func (m *method) RequireGreaterThanOrEqual(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectGreaterThanOrEqual", v1, v2, compGreaterOrEqualThan, false, true))
	return m
}

//go:noinline
func (m *method) AssertNotLessThan(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectNotLessThan", v1, v2, compLessThan, true, false))
	return m
}

//go:noinline
func (m *method) RequireNotLessThan(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectNotLessThan", v1, v2, compLessThan, true, true))
	return m
}

//go:noinline
func (m *method) AssertNotGreaterThan(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectNotGreaterThan", v1, v2, compGreaterThan, true, false))
	return m
}

//go:noinline
func (m *method) RequireNotGreaterThan(v1, v2 any) Method_ {
	m.addPostExpectation(newComparator(1, "ExpectNotGreaterThan", v1, v2, compGreaterThan, true, true))
	return m
}

//go:noinline
func (m *method) AssertMatch(value any, regex string) Method_ {
	m.addPostExpectation(&match{
		value: value,
		regex: regex,
		frame: framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireMatch(value any, regex string) Method_ {
	m.addPostExpectation(&match{
		value:             value,
		regex:             regex,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertContains(value any, s string) Method_ {
	m.addPostExpectation(&contains{
		value:    value,
		contains: s,
		frame:    framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireContains(value any, s string) Method_ {
	m.addPostExpectation(&contains{
		value:             value,
		contains:          s,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertType(value any, typ Type_) Method_ {
	m.addPostExpectation(&matchType{
		value: value,
		typ:   typ,
		frame: framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireType(value any, typ Type_) Method_ {
	m.addPostExpectation(&matchType{
		value:             value,
		typ:               typ,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertNil(value any) Method_ {
	m.addPostExpectation(&nilCheck{
		value: value,
		frame: framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireNil(value any) Method_ {
	m.addPostExpectation(&nilCheck{
		value:             value,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertNotNil(value any) Method_ {
	m.addPostExpectation(&notNilCheck{
		value: value,
		frame: framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireNotNil(value any) Method_ {
	m.addPostExpectation(&notNilCheck{
		value:             value,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertLen(value any, length int) Method_ {
	m.addPostExpectation(&lenCheck{
		value:  value,
		length: length,
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireLen(value any, length int) Method_ {
	m.addPostExpectation(&lenCheck{
		value:             value,
		length:            length,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertHasProperties(value any, propertyNames ...string) Method_ {
	m.addPostExpectation(&propertiesCheck{
		value:      value,
		properties: propertyNames,
		frame:      framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireHasProperties(value any, propertyNames ...string) Method_ {
	m.addPostExpectation(&propertiesCheck{
		value:             value,
		properties:        propertyNames,
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
	return m
}

//go:noinline
func (m *method) AssertOnlyHasProperties(value any, propertyNames ...string) Method_ {
	m.addPostExpectation(&propertiesCheck{
		value:      value,
		properties: propertyNames,
		only:       true,
		frame:      framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireOnlyHasProperties(value any, propertyNames ...string) Method_ {
	m.addPostExpectation(&propertiesCheck{
		value:             value,
		properties:        propertyNames,
		frame:             framing.NewFrame(0),
		only:              true,
		commonExpectation: commonExpectation{required: true},
	})
	return m
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

func (m *method) RequestMarshal(fn func(ctx Context, body any) ([]byte, error)) Method_ {
	m.requestMarshal = fn
	return m
}

func (m *method) ResponseUnmarshal(fn func(response *http.Response) (any, error)) Method_ {
	m.responseUnmarshal = fn
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

//go:noinline
func (m *method) AssertMockServiceCalled(svcName string, path string, method MethodName) Method_ {
	m.addPostExpectation(&expectMockCall{
		name:   svcName,
		path:   path,
		method: strings.ToUpper(string(method)),
		frame:  framing.NewFrame(0),
	})
	return m
}

//go:noinline
func (m *method) RequireMockServiceCalled(svcName string, path string, method MethodName) Method_ {
	m.addPostExpectation(&expectMockCall{
		name:              svcName,
		path:              path,
		method:            strings.ToUpper(string(method)),
		frame:             framing.NewFrame(0),
		commonExpectation: commonExpectation{required: true},
	})
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
func (m *method) If(when When, condition any, ops ...Runnable) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &conditional{
			condition: condition,
			ops:       ops,
			frame:     framing.NewFrame(0),
		})
	} else {
		m.addPostCapture(&conditional{
			condition: condition,
			ops:       ops,
			frame:     framing.NewFrame(0),
		})
	}
	return m
}

func (m *method) Run(ctx Context) error {
	ctx.setCurrentMethod(m)
	if m.preRun(ctx) {
		if request, ok := m.buildRequest(ctx); ok {
			if m.preRequestRun(ctx) {
				if response, ok := ctx.doRequest(request); ok {
					if m.unmarshalResponseBody(ctx, response) {
						m.postRun(ctx)
					}
				}
			}
		}
	}
	return nil
}

func (m *method) preRun(ctx Context) bool {
	for _, c := range m.preCaptures {
		if c != nil {
			if err := c.Run(ctx); err != nil {
				ctx.reportFailure(err)
				return false
			}
		}
	}
	return true
}

func (m *method) preRequestRun(ctx Context) bool {
	for _, c := range m.authFns {
		if c != nil {
			if err := c.Run(ctx); err != nil {
				ctx.reportFailure(err)
				return false
			}
		}
	}
	return true
}

func (m *method) postRun(ctx Context) {
	ok := true
	lastExp := 0
	for _, po := range m.postOps {
		if po.isExpectation {
			lastExp = po.index
			exp := m.expectations[lastExp]
			if exp != nil {
				if unmet, err := exp.Met(ctx); err != nil {
					ctx.reportFailure(err)
					ok = false
					break
				} else if unmet != nil {
					ctx.reportUnmet(exp, unmet)
					if m.failFast || exp.IsRequired() {
						ok = false
						break
					}
				} else {
					ctx.reportMet(exp)
				}
			}
		} else {
			c := m.postCaptures[po.index]
			if c != nil {
				if rErr := c.Run(ctx); rErr != nil {
					ctx.reportFailure(rErr)
					ok = false
					break
				}
			}
		}
	}
	if !ok {
		for s := lastExp + 1; s < len(m.expectations); s++ {
			ctx.reportSkipped(m.expectations[s])
		}
	}
}

func (m *method) unmarshalResponseBody(ctx Context, res *http.Response) bool {
	if res.Body != nil {
		var body any
		var err error
		if m.responseUnmarshal != nil {
			body, err = m.responseUnmarshal(res)
		} else {
			decoder := json.NewDecoder(res.Body)
			decoder.UseNumber()
			if err = decoder.Decode(&body); err == nil {
				body, err = normalizeBody(body)
			} else if err == io.EOF {
				body = nil
				err = nil
			}
		}
		if err != nil {
			ctx.reportFailure(err)
			return false
		}
		ctx.setCurrentBody(body)
	} else {
		ctx.setCurrentBody(nil)
	}
	return true
}

func normalizeBody(body any) (any, error) {
	if body == nil {
		return nil, nil
	}
	switch bt := body.(type) {
	case json.Number:
		return normalizeBodyJsonNumber(bt)
	case map[string]any:
		if err := normalizeBodyMap(bt); err != nil {
			return nil, err
		}
	case []any:
		if err := normalizeBodySlice(bt); err != nil {
			return nil, err
		}
	}
	return body, nil
}

func normalizeBodyJsonNumber(jn json.Number) (any, error) {
	if s := strings.Map(func(r rune) rune {
		if r == '-' || (r >= '0' && r <= '9') {
			return -1
		}
		return r
	}, jn.String()); s == "" {
		return jn.Int64()
	}
	return jn.Float64()
}

func normalizeBodyMap(m map[string]any) error {
	for k := range maps.Keys(m) {
		v := m[k]
		switch vt := v.(type) {
		case json.Number:
			if nv, err := normalizeBodyJsonNumber(vt); err == nil {
				m[k] = nv
			} else {
				return err
			}
		case map[string]any:
			if err := normalizeBodyMap(vt); err != nil {
				return err
			}
		case []any:
			if err := normalizeBodySlice(vt); err != nil {
				return err
			}
		}
	}
	return nil
}

func normalizeBodySlice(sl []any) error {
	for i, v := range sl {
		switch vt := v.(type) {
		case json.Number:
			if nv, err := normalizeBodyJsonNumber(vt); err == nil {
				sl[i] = nv
			} else {
				return err
			}
		case map[string]any:
			if err := normalizeBodyMap(vt); err != nil {
				return err
			}
		case []any:
			if err := normalizeBodySlice(vt); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *method) buildRequest(ctx Context) (request *http.Request, ok bool) {
	const contentType = "Content-Type"
	var url string
	var err error
	defer func() {
		if ok = err == nil; !ok {
			ctx.reportFailure(err)
		}
	}()
	if url, err = m.buildRequestUrl(ctx); err == nil {
		var body io.Reader
		if body, err = m.buildRequestBody(ctx); err == nil {
			meth := string(m.method)
			if meth == "" {
				meth = http.MethodGet
			}
			if request, err = http.NewRequestWithContext(ctx.Ctx(), meth, url, body); err == nil {
				seenContentType := false
				for h, v := range m.headers {
					var av any
					if av, err = ResolveValue(v, ctx); err == nil {
						request.Header.Set(h, fmt.Sprintf("%v", av))
						seenContentType = (h == contentType) || seenContentType
					} else {
						return
					}
				}
				if !seenContentType {
					request.Header.Set("Content-Type", "application/json")
				}
				for ck := range m.useCookies {
					if c := ctx.GetCookie(ck); c != nil {
						request.AddCookie(c)
					}
				}
			}
		}
	}
	return
}

func (m *method) buildRequestBody(ctx Context) (body io.Reader, err error) {
	if m.body != nil {
		var av any
		if av, err = ResolveValue(m.body, ctx); err == nil {
			var data []byte
			if m.requestMarshal != nil {
				data, err = m.requestMarshal(ctx, av)
			} else {
				data, err = json.Marshal(av)
			}
			if err == nil && len(data) > 0 {
				body = bytes.NewReader(data)
			}
		}
	}
	return
}

func (m *method) buildRequestUrl(ctx Context) (url string, err error) {
	u := ctx.CurrentUrl()
	var template urit.Template
	if template, err = urit.NewTemplate(u); err == nil {
		var pps pathParams
		if pps, err = m.pathParams.resolve(ctx); err == nil {
			if url, err = template.PathFrom(pps); err == nil {
				var q string
				if q, err = m.queryParams.encode(ctx); err == nil {
					url = ctx.Host() + url + q
				}
			}
		}
	}
	return
}

func (m *method) String() string {
	return fmt.Sprintf("%s %q", string(m.method), m.desc)
}

type MethodName string

func (m MethodName) Normalize() MethodName {
	return MethodName(strings.ToUpper(string(m)))
}

const (
	GET     MethodName = http.MethodGet
	HEAD    MethodName = http.MethodHead
	POST    MethodName = http.MethodPost
	PUT     MethodName = http.MethodPut
	PATCH   MethodName = http.MethodPatch
	DELETE  MethodName = http.MethodDelete
	OPTIONS MethodName = http.MethodOptions
	CONNECT MethodName = http.MethodConnect
	TRACE   MethodName = http.MethodTrace
)
