package marrow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-andiamo/urit"
	"io"
	"maps"
	"net/http"
	"strings"
)

type Method_ interface {
	Method() MethodName
	Description() string

	Authorize(func(ctx Context) error) Method_
	QueryParam(name string, values ...any) Method_
	PathParam(value any) Method_
	RequestHeader(name string, value any) Method_
	RequestBody(value any) Method_
	UseCookie(name string) Method_
	SetCookie(cookie *http.Cookie) Method_
	StoreCookie(name string) Method_

	ExpectOK() Method_
	ExpectStatus(status any) Method_
	Capture(op BeforeAfter_) Method_
	CaptureFunc(when When, fn func(Context) error) Method_
	Expect(exp Expectation) Method_
	ExpectEqual(v1, v2 any) Method_
	ExpectNotEqual(v1, v2 any) Method_
	ExpectLessThan(v1, v2 any) Method_
	ExpectLessThanOrEqual(v1, v2 any) Method_
	ExpectGreaterThan(v1, v2 any) Method_
	ExpectGreaterThanOrEqual(v1, v2 any) Method_
	ExpectNotLessThan(v1, v2 any) Method_
	ExpectNotGreaterThan(v1, v2 any) Method_
	ExpectMatch(value any, regex string) Method_
	ExpectType(value any, typ Type_) Method_
	ExpectNil(value any) Method_
	ExpectNotNil(value any) Method_
	SetVar(when When, name string, value any) Method_
	ClearVars(when When) Method_
	DbInsert(when When, tableName string, row Columns) Method_
	DbExec(when When, query string, args ...any) Method_
	DbClearTables(when When, tableNames ...string) Method_

	RequestMarshal(fn func(ctx Context, body any) ([]byte, error)) Method_
	ResponseUnmarshal(fn func(response *http.Response) (any, error)) Method_
	Runnable
	fmt.Stringer
}

//go:noinline
func Method(m MethodName, desc string, ops ...BeforeAfter_) Method_ {
	result := &method{
		desc:         desc,
		frame:        frame(0),
		method:       m,
		queryParams:  queryParams{},
		pathParams:   pathParams{},
		headers:      make(map[string]any),
		expectations: make([]Expectation, 0),
		useCookies:   make(map[string]struct{}),
	}
	for _, op := range ops {
		if op != nil {
			if op.When() == Before {
				result.preCaptures = append(result.preCaptures, op)
			} else {
				result.postCaptures = append(result.postCaptures, op)
			}
		}
	}
	return result
}

type method struct {
	desc              string
	frame             *Frame
	method            MethodName
	pathParams        pathParams
	queryParams       queryParams
	headers           map[string]any
	body              any
	preCaptures       []Runnable
	postCaptures      []Runnable
	expectations      []Expectation
	useCookies        map[string]struct{}
	requestMarshal    func(ctx Context, body any) ([]byte, error)
	responseUnmarshal func(response *http.Response) (any, error)
}

func (m *method) Method() MethodName {
	return m.method
}

func (m *method) Description() string {
	return m.desc
}

func (m *method) Frame() *Frame {
	return m.frame
}

//go:noinline
func (m *method) Authorize(fn func(ctx Context) error) Method_ {
	if fn != nil {
		m.preCaptures = append(m.preCaptures, &userDefinedCapture{
			name:  "Authorize",
			fn:    fn,
			frame: frame(0),
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
			frame:  frame(0),
		})
		m.useCookies[cookie.Name] = struct{}{}
	}
	return m
}

//go:noinline
func (m *method) StoreCookie(name string) Method_ {
	m.postCaptures = append(m.postCaptures, &storeCookie{
		name:  name,
		frame: frame(0),
	})
	return m
}

//go:noinline
func (m *method) ExpectOK() Method_ {
	m.expectations = append(m.expectations, newExpectOk(1))
	return m
}

//go:noinline
func (m *method) ExpectStatus(status any) Method_ {
	m.expectations = append(m.expectations, newExpectStatusCode(1, status))
	return m
}

//go:noinline
func (m *method) Capture(op BeforeAfter_) Method_ {
	if op != nil {
		if op.When() == Before {
			m.preCaptures = append(m.preCaptures, op)
		} else {
			m.postCaptures = append(m.postCaptures, op)
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
				frame: frame(0),
			})
		} else {
			m.preCaptures = append(m.preCaptures, &userDefinedCapture{
				fn:    fn,
				frame: frame(0),
			})
		}
	}
	return m
}

//go:noinline
func (m *method) Expect(exp Expectation) Method_ {
	m.expectations = append(m.expectations, exp)
	return m
}

//go:noinline
func (m *method) ExpectEqual(v1, v2 any) Method_ {
	m.expectations = append(m.expectations, newComparator(1, "ExpectEqual", v1, v2, compEqual, false))
	return m
}

//go:noinline
func (m *method) ExpectNotEqual(v1, v2 any) Method_ {
	m.expectations = append(m.expectations, newComparator(1, "ExpectNotEqual", v1, v2, compEqual, true))
	return m
}

//go:noinline
func (m *method) ExpectLessThan(v1, v2 any) Method_ {
	m.expectations = append(m.expectations, newComparator(1, "ExpectLessThan", v1, v2, compLessThan, false))
	return m
}

//go:noinline
func (m *method) ExpectLessThanOrEqual(v1, v2 any) Method_ {
	m.expectations = append(m.expectations, newComparator(1, "ExpectLessThanOrEqual", v1, v2, compLessOrEqualThan, false))
	return m
}

//go:noinline
func (m *method) ExpectGreaterThan(v1, v2 any) Method_ {
	m.expectations = append(m.expectations, newComparator(1, "ExpectGreaterThan", v1, v2, compGreaterThan, false))
	return m
}

//go:noinline
func (m *method) ExpectGreaterThanOrEqual(v1, v2 any) Method_ {
	m.expectations = append(m.expectations, newComparator(1, "ExpectGreaterThanOrEqual", v1, v2, compGreaterOrEqualThan, false))
	return m
}

//go:noinline
func (m *method) ExpectNotLessThan(v1, v2 any) Method_ {
	m.expectations = append(m.expectations, newComparator(1, "ExpectNotLessThan", v1, v2, compLessThan, true))
	return m
}

//go:noinline
func (m *method) ExpectNotGreaterThan(v1, v2 any) Method_ {
	m.expectations = append(m.expectations, newComparator(1, "ExpectNotGreaterThan", v1, v2, compGreaterThan, true))
	return m
}

//go:noinline
func (m *method) ExpectMatch(value any, regex string) Method_ {
	m.expectations = append(m.expectations, &match{
		value: value,
		regex: regex,
		frame: frame(0),
	})
	return m
}

//go:noinline
func (m *method) ExpectType(value any, typ Type_) Method_ {
	m.expectations = append(m.expectations, &matchType{
		value: value,
		typ:   typ,
		frame: frame(0),
	})
	return m
}

//go:noinline
func (m *method) ExpectNil(value any) Method_ {
	m.expectations = append(m.expectations, &nilCheck{
		value: value,
		frame: frame(0),
	})
	return m
}

//go:noinline
func (m *method) ExpectNotNil(value any) Method_ {
	m.expectations = append(m.expectations, &notNilCheck{
		value: value,
		frame: frame(0),
	})
	return m
}

//go:noinline
func (m *method) SetVar(when When, name string, value any) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &setVar{
			name:  name,
			value: value,
			frame: frame(0),
		})
	} else {
		m.postCaptures = append(m.postCaptures, &setVar{
			name:  name,
			value: value,
			frame: frame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) ClearVars(when When) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &clearVars{
			frame: frame(0),
		})
	} else {
		m.postCaptures = append(m.postCaptures, &clearVars{
			frame: frame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) DbInsert(when When, tableName string, row Columns) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &dbInsert{
			tableName: tableName,
			row:       row,
			frame:     frame(0),
		})
	} else {
		m.postCaptures = append(m.postCaptures, &dbInsert{
			tableName: tableName,
			row:       row,
			frame:     frame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) DbExec(when When, query string, args ...any) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &dbExec{
			query: query,
			args:  args,
			frame: frame(0),
		})
	} else {
		m.postCaptures = append(m.postCaptures, &dbExec{
			query: query,
			args:  args,
			frame: frame(0),
		})
	}
	return m
}

//go:noinline
func (m *method) DbClearTables(when When, tableNames ...string) Method_ {
	if when == Before {
		for _, tableName := range tableNames {
			m.preCaptures = append(m.preCaptures, &dbClearTable{
				tableName: tableName,
				frame:     frame(0),
			})
		}
	} else {
		for _, tableName := range tableNames {
			m.postCaptures = append(m.postCaptures, &dbClearTable{
				tableName: tableName,
				frame:     frame(0),
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

func (m *method) Run(ctx Context) error {
	ctx.setCurrentMethod(m)
	for _, c := range m.preCaptures {
		if c != nil {
			if err := c.Run(ctx); err != nil {
				ctx.reportFailure(err)
				return nil
			}
		}
	}
	request, err := m.buildRequest(ctx)
	if err != nil {
		ctx.reportFailure(err)
		return nil
	}
	if res, ok := ctx.doRequest(request); ok {
		if m.unmarshalResponseBody(ctx, res) {
			for _, c := range m.postCaptures {
				if c != nil {
					if err := c.Run(ctx); err != nil {
						ctx.reportFailure(err)
						return nil
					}
				}
			}
			for _, exp := range m.expectations {
				if exp != nil {
					if unmet, err := exp.Met(ctx); err != nil {
						ctx.reportFailure(err)
						return nil
					} else if unmet != nil {
						ctx.reportUnmet(exp, unmet)
					} else {
						ctx.reportMet(exp)
					}
				}
			}
		}
	}
	return nil
}

func (m *method) unmarshalResponseBody(ctx Context, res *http.Response) bool {
	if res.Body != nil {
		if m.responseUnmarshal != nil {
			if body, err := m.responseUnmarshal(res); err == nil {
				ctx.setCurrentBody(body)
			} else {
				ctx.reportFailure(err)
				return false
			}
		} else {
			var body any
			var err error
			decoder := json.NewDecoder(res.Body)
			decoder.UseNumber()
			if err = decoder.Decode(&body); err != nil {
				ctx.reportFailure(err)
				return false
			}
			if body, err = normalizeBody(body); err != nil {
				ctx.reportFailure(err)
				return false
			}
			ctx.setCurrentBody(body)
		}
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

func (m *method) buildRequest(ctx Context) (request *http.Request, err error) {
	const contentType = "Content-Type"
	var url string
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
					if av, err := ResolveValue(v, ctx); err == nil {
						request.Header.Set(h, fmt.Sprintf("%v", av))
						seenContentType = (h == contentType) || seenContentType
					} else {
						return nil, err
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
