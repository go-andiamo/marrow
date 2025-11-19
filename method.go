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

	MethodExpectations
	MethodCaptures

	// If runs the operations when the condition arg is met
	//
	// Notes:
	//   * the condition arg can be a bool value (or value that resolves to a bool) or an Expectation (e.g. ExpectEqual, ExpectNotEqual, etc.)
	//   * if the condition arg is an Expectation, and the expectation is unmet, this does not report a failure or unmet, instead the operations are just not performed
	//   * any condition that is not a bool or Expectation will cause an error during tests
	//   * the operations arg can be anything Runnable - any of them that are an Expectation, is run as an expectation (and treated as required) and any unmet or failure errors will be reported
	If(when When, condition any, operations ...Runnable) Method_
	// IfNot runs the operations when the condition arg is not met
	//
	// Notes:
	//   * the condition arg can be a bool value (or value that resolves to a bool) or an Expectation (e.g. ExpectEqual, ExpectNotEqual, etc.)
	//   * if the condition arg is an Expectation, and the expectation is unmet, this does not report a failure or unmet, instead the operations are just not performed
	//   * any condition that is not a bool or Expectation will cause an error during tests
	//   * the operations arg can be anything Runnable - any of them that are an Expectation, is run as an expectation (and treated as required) and any unmet or failure errors will be reported
	IfNot(when When, condition any, operations ...Runnable) Method_
	// FailFast instructs the method to fail on unmet assertions
	//
	// i.e. treat all `Assert...()` as `Require...()`
	FailFast() Method_

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

func (m *method) RequestMarshal(fn func(ctx Context, body any) ([]byte, error)) Method_ {
	m.requestMarshal = fn
	return m
}

func (m *method) ResponseUnmarshal(fn func(response *http.Response) (any, error)) Method_ {
	m.responseUnmarshal = fn
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

//go:noinline
func (m *method) IfNot(when When, condition any, ops ...Runnable) Method_ {
	if when == Before {
		m.preCaptures = append(m.preCaptures, &conditional{
			condition: condition,
			not:       true,
			ops:       ops,
			frame:     framing.NewFrame(0),
		})
	} else {
		m.addPostCapture(&conditional{
			condition: condition,
			not:       true,
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
				ctx.setCurrentRequest(request)
				if response, ok := ctx.doRequest(); ok {
					if m.unmarshalResponseBody(ctx, response) {
						m.postRun(ctx)
					}
				}
			}
		}
	}
	ctx.setCurrentMethod(nil)
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
