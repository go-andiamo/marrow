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
	Expect(exp Expectation) Method_

	AssertOK() Method_
	AssertCreated() Method_
	AssertAccepted() Method_
	AssertNoContent() Method_
	AssertBadRequest() Method_
	AssertUnauthorized() Method_
	AssertForbidden() Method_
	AssertNotFound() Method_
	AssertConflict() Method_
	AssertGone() Method_
	AssertUnprocessableEntity() Method_
	AssertStatus(status any) Method_
	AssertFunc(fn func(Context) (unmet error, err error)) Method_
	AssertEqual(v1, v2 any) Method_
	AssertNotEqual(v1, v2 any) Method_
	AssertLessThan(v1, v2 any) Method_
	AssertLessThanOrEqual(v1, v2 any) Method_
	AssertGreaterThan(v1, v2 any) Method_
	AssertGreaterThanOrEqual(v1, v2 any) Method_
	AssertNotLessThan(v1, v2 any) Method_
	AssertNotGreaterThan(v1, v2 any) Method_
	AssertMatch(value any, regex string) Method_
	AssertType(value any, typ Type_) Method_
	AssertNil(value any) Method_
	AssertNotNil(value any) Method_
	AssertLen(value any, length int) Method_
	AssertHasProperties(value any, propertyNames ...string) Method_
	AssertOnlyHasProperties(value any, propertyNames ...string) Method_

	RequireOK() Method_
	RequireCreated() Method_
	RequireAccepted() Method_
	RequireNoContent() Method_
	RequireBadRequest() Method_
	RequireUnauthorized() Method_
	RequireForbidden() Method_
	RequireNotFound() Method_
	RequireConflict() Method_
	RequireGone() Method_
	RequireUnprocessableEntity() Method_
	RequireStatus(status any) Method_
	RequireFunc(fn func(Context) (unmet error, err error)) Method_
	RequireEqual(v1, v2 any) Method_
	RequireNotEqual(v1, v2 any) Method_
	RequireLessThan(v1, v2 any) Method_
	RequireLessThanOrEqual(v1, v2 any) Method_
	RequireGreaterThan(v1, v2 any) Method_
	RequireGreaterThanOrEqual(v1, v2 any) Method_
	RequireNotLessThan(v1, v2 any) Method_
	RequireNotGreaterThan(v1, v2 any) Method_
	RequireMatch(value any, regex string) Method_
	RequireType(value any, typ Type_) Method_
	RequireNil(value any) Method_
	RequireNotNil(value any) Method_
	RequireLen(value any, length int) Method_
	RequireHasProperties(value any, propertyNames ...string) Method_
	RequireOnlyHasProperties(value any, propertyNames ...string) Method_

	// FailFast instructs the method to fail on unmet assertions
	//
	// i.e. treat all `Assert...()` as `Require...()`
	FailFast() Method_
}

type methodMockService interface {
	MockServicesClearAll(when When) Method_
	MockServiceClear(when When, svcName string) Method_
	MockServiceCall(svcName string, path string, method MethodName, responseStatus int, responseBody any, headers ...any) Method_
	AssertMockServiceCalled(svcName string, path string, method MethodName) Method_
	RequireMockServiceCalled(svcName string, path string, method MethodName) Method_
}

type Method_ interface {
	common.Method

	Authorize(func(ctx Context) error) Method_
	QueryParam(name string, values ...any) Method_
	PathParam(value any) Method_
	RequestHeader(name string, value any) Method_
	RequestBody(value any) Method_
	UseCookie(name string) Method_
	SetCookie(cookie *http.Cookie) Method_
	StoreCookie(name string) Method_

	methodExpectations
	methodMockService

	Capture(op BeforeAfter_) Method_
	CaptureFunc(when When, fn func(Context) error) Method_
	SetVar(when When, name string, value any) Method_
	ClearVars(when When) Method_
	DbInsert(when When, dbName string, tableName string, row Columns) Method_
	DbExec(when When, dbName string, query string, args ...any) Method_
	DbClearTables(when When, dbName string, tableNames ...string) Method_
	Wait(when When, ms int) Method_

	RequestMarshal(fn func(ctx Context, body any) ([]byte, error)) Method_
	ResponseUnmarshal(fn func(response *http.Response) (any, error)) Method_
	Runnable
	fmt.Stringer
}

//go:noinline
func Method(m MethodName, desc string, ops ...BeforeAfter_) Method_ {
	result := &method{
		desc:        desc,
		frame:       framing.NewFrame(0),
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
		m.preCaptures = append(m.preCaptures, &userDefinedCapture{
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
func (m *method) Capture(op BeforeAfter_) Method_ {
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
func (m *method) SetVar(when When, name string, value any) Method_ {
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

func (m *method) Run(ctx Context) error {
	ctx.setCurrentMethod(m)
	if m.preRun(ctx) {
		if request, ok := m.buildRequest(ctx); ok {
			if response, ok := ctx.doRequest(request); ok {
				if m.unmarshalResponseBody(ctx, response) {
					m.postRun(ctx)
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
