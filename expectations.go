package marrow

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/framing"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Expectation interface {
	common.Expectation
	Runnable
	Met(ctx Context) (unmet error, err error)
	IsRequired() bool
}

type commonExpectation struct {
	required bool
}

func (e commonExpectation) IsRequired() bool {
	return e.required
}

func (e commonExpectation) Run(ctx Context) error {
	panic("direct call to .Run() on Expectation")
}

//go:noinline
func ExpectationFunc(fn func(ctx Context) (unmet error, err error)) Expectation {
	return &expectation{
		fn:    fn,
		frame: framing.NewFrame(0),
	}
}

type expectation struct {
	fn    func(ctx Context) (unmet error, err error)
	frame *framing.Frame
	commonExpectation
}

var _ Expectation = (*expectation)(nil)

func (e *expectation) Name() string {
	return "(User Defined Expectation)"
}

func (e *expectation) Met(ctx Context) (unmet error, err error) {
	if e.fn != nil {
		unmet, err = e.fn(ctx)
		if unmet != nil {
			unmet = &unmetError{
				msg:   "user defined expectation failed",
				name:  e.Name(),
				cause: unmet,
				frame: e.frame,
			}
		}
	}
	return
}

func (e *expectation) Frame() *framing.Frame {
	return e.frame
}

type expectStatusCode struct {
	name   string
	expect any
	frame  *framing.Frame
	commonExpectation
}

var _ Expectation = (*expectStatusCode)(nil)

// ExpectOK asserts the response status code is 200 "OK"
//
//go:noinline
func ExpectOK() Expectation {
	return &expectStatusCode{
		name:   "Expect OK",
		expect: http.StatusOK,
		frame:  framing.NewFrame(0),
	}
}

// ExpectCreated asserts the response status code is 201 "Created"
//
//go:noinline
func ExpectCreated() Expectation {
	return &expectStatusCode{
		name:   "Expect Created",
		expect: http.StatusCreated,
		frame:  framing.NewFrame(0),
	}
}

// ExpectAccepted asserts the response status code is 202 "Accepted"
//
//go:noinline
func ExpectAccepted() Expectation {
	return &expectStatusCode{
		name:   "Expect Accepted",
		expect: http.StatusAccepted,
		frame:  framing.NewFrame(0),
	}
}

// ExpectNoContent asserts the response status code is 204 "No Content"
//
//go:noinline
func ExpectNoContent() Expectation {
	return &expectStatusCode{
		name:   "Expect No Content",
		expect: http.StatusNoContent,
		frame:  framing.NewFrame(0),
	}
}

// ExpectBadRequest asserts the response status code is 400 "Bad Request"
//
//go:noinline
func ExpectBadRequest() Expectation {
	return &expectStatusCode{
		name:   "Expect Bad Request",
		expect: http.StatusBadRequest,
		frame:  framing.NewFrame(0),
	}
}

// ExpectUnauthorized asserts the response status code is 401 "Unauthorized"
//
//go:noinline
func ExpectUnauthorized() Expectation {
	return &expectStatusCode{
		name:   "Expect Unauthorized",
		expect: http.StatusUnauthorized,
		frame:  framing.NewFrame(0),
	}
}

// ExpectForbidden asserts the response status code is 403 "Forbidden"
//
//go:noinline
func ExpectForbidden() Expectation {
	return &expectStatusCode{
		name:   "Expect Forbidden",
		expect: http.StatusForbidden,
		frame:  framing.NewFrame(0),
	}
}

// ExpectNotFound asserts the response status code is 404 "Not Found"
//
//go:noinline
func ExpectNotFound() Expectation {
	return &expectStatusCode{
		name:   "Expect Not Found",
		expect: http.StatusNotFound,
		frame:  framing.NewFrame(0),
	}
}

// ExpectConflict asserts the response status code is 409 "Conflict"
//
//go:noinline
func ExpectConflict() Expectation {
	return &expectStatusCode{
		name:   "Expect Conflict",
		expect: http.StatusConflict,
		frame:  framing.NewFrame(0),
	}
}

// ExpectGone asserts the response status code is 410 "Gone"
//
//go:noinline
func ExpectGone() Expectation {
	return &expectStatusCode{
		name:   "Expect Gone",
		expect: http.StatusGone,
		frame:  framing.NewFrame(0),
	}
}

// ExpectUnprocessableEntity asserts the response status code is 422 "Unprocessable Entity"
//
//go:noinline
func ExpectUnprocessableEntity() Expectation {
	return &expectStatusCode{
		name:   "Expect Unprocessable Entity",
		expect: http.StatusUnprocessableEntity,
		frame:  framing.NewFrame(0),
	}
}

// ExpectStatus asserts the response status code is the status supplied
//
//go:noinline
func ExpectStatus(status any) Expectation {
	return &expectStatusCode{
		name:   "Expect Status Code",
		expect: status,
		frame:  framing.NewFrame(0),
	}
}

func (e *expectStatusCode) Name() string {
	return e.name
}

func (e *expectStatusCode) Met(ctx Context) (unmet error, err error) {
	ev := OperandValue{Original: e.expect}
	if ev.Resolved, err = ResolveValue(ev.Original, ctx); err != nil {
		return
	}
	switch evt := ev.Resolved.(type) {
	case string:
		ev.Coerced, ev.CoercionError = strconv.Atoi(evt)
	case int:
		ev.Coerced = evt
	case int64:
		ev.Coerced = int(evt)
	default:
		ev.CoercionError = fmt.Errorf("cannot coerce %T to %T", ev.Resolved, 0)
	}
	sc := -1
	if response := ctx.CurrentResponse(); response != nil {
		sc = response.StatusCode
	}
	if ev.CoercionError == nil && sc == ev.Coerced.(int) {
		return nil, nil
	}
	var msg string
	if ev.CoercionError == nil {
		evs := Status(ev.Coerced.(int))
		msg = "expected status code " + evs.stringify()
		ev.Resolved = evs
	} else {
		msg = "expected status code - cannot be compared"
	}
	unmet = &unmetError{
		msg:      msg,
		name:     e.Name(),
		actual:   OperandValue{Original: sc, Resolved: Status(sc)},
		expected: ev,
		frame:    e.frame,
	}
	return
}

type Status int

func (s Status) stringify() string {
	if st := http.StatusText(int(s)); st != "" {
		return fmt.Sprintf("%d %q", int(s), st)
	}
	return strconv.Itoa(int(s))
}

func (e *expectStatusCode) Frame() *framing.Frame {
	return e.frame
}

type match struct {
	value any
	regex string
	rx    *regexp.Regexp
	frame *framing.Frame
	commonExpectation
}

var _ Expectation = (*match)(nil)

// ExpectMatch asserts that the value matches the supplied regex
//
// when attempting to match against the regex, the value (or resolved value) is "stringified"
//
// values can be any of:
//   - primitive type of string, bool, int, int64, float64
//   - decimal.Decimal
//   - or anything that is resolvable...
//
// examples of resolvable values are: Var, Body, BodyPath, Query, QueryRows, JsonPath, JsonTraverse,
// StatusCode, ResponseCookie, ResponseHeader, JSON, JSONArray, TemplateString,
//
//go:noinline
func ExpectMatch(value any, regex string) Expectation {
	return &match{
		value: value,
		regex: regex,
		frame: framing.NewFrame(0),
	}
}

func (m *match) Name() string {
	return fmt.Sprintf("Expect match: %q", m.regex)
}

func (m *match) Met(ctx Context) (unmet error, err error) {
	ov := OperandValue{Original: m.value}
	if m.rx == nil {
		if m.rx, err = regexp.Compile(m.regex); err != nil {
			return
		}
	}
	if ov.Resolved, err = ResolveValue(m.value, ctx); err == nil {
		switch avt := ov.Resolved.(type) {
		case string:
			ov.Coerced = avt
		default:
			to := reflect.TypeOf(ov.Resolved)
			if to.Kind() == reflect.Map || to.Kind() == reflect.Slice || to.Kind() == reflect.Struct {
				if data, mErr := json.Marshal(ov.Resolved); mErr == nil {
					ov.Coerced = string(data)
				} else {
					ov.CoercionError = mErr
					unmet = &unmetError{
						msg:    fmt.Sprintf("expected match %q", m.regex),
						name:   m.Name(),
						actual: ov,
						cause:  mErr,
						frame:  m.frame,
					}
					return
				}
			} else {
				ov.Coerced = fmt.Sprintf("%v", ov.Resolved)
			}
		}
		if ok := m.rx.MatchString(ov.Coerced.(string)); !ok {
			unmet = &unmetError{
				msg:    fmt.Sprintf("expected match %q", m.regex),
				name:   m.Name(),
				actual: ov,
				frame:  m.frame,
			}
		}
	}
	return
}

func (m *match) Frame() *framing.Frame {
	return m.frame
}

type contains struct {
	value    any
	contains string
	rx       *regexp.Regexp
	frame    *framing.Frame
	commonExpectation
}

var _ Expectation = (*contains)(nil)

// ExpectContains asserts that the value contains a substring
//
// when attempting to check contains, the value (or resolved value) is "stringified"
//
//go:noinline
func ExpectContains(value any, s string) Expectation {
	return &contains{
		value:    value,
		contains: s,
		frame:    framing.NewFrame(0),
	}
}

func (c *contains) Name() string {
	return fmt.Sprintf("Expect contains: %q", c.contains)
}

func (c *contains) Frame() *framing.Frame {
	return c.frame
}

func (c *contains) Met(ctx Context) (unmet error, err error) {
	ov := OperandValue{Original: c.value}
	if ov.Resolved, err = ResolveValue(c.value, ctx); err == nil {
		switch avt := ov.Resolved.(type) {
		case string:
			ov.Coerced = avt
		default:
			to := reflect.TypeOf(ov.Resolved)
			if to.Kind() == reflect.Map || to.Kind() == reflect.Slice || to.Kind() == reflect.Struct {
				if data, mErr := json.Marshal(ov.Resolved); mErr == nil {
					ov.Coerced = string(data)
				} else {
					ov.CoercionError = mErr
					unmet = &unmetError{
						msg:    fmt.Sprintf("expected contains %q", c.contains),
						name:   c.Name(),
						actual: ov,
						cause:  mErr,
						frame:  c.frame,
					}
					return
				}
			} else {
				ov.Coerced = fmt.Sprintf("%v", ov.Resolved)
			}
		}
		if ok := strings.Contains(ov.Coerced.(string), c.contains); !ok {
			unmet = &unmetError{
				msg:    fmt.Sprintf("expected contains %q", c.contains),
				name:   c.Name(),
				actual: ov,
				frame:  c.frame,
			}
		}
	}
	return
}

type matchType struct {
	value any
	typ   Type_
	frame *framing.Frame
	commonExpectation
}

var _ Expectation = (*matchType)(nil)

// ExpectType asserts that the value (or resolved value) is of the supplied type
//
//go:noinline
func ExpectType(value any, typ Type_) Expectation {
	return &matchType{
		value: value,
		typ:   typ,
		frame: framing.NewFrame(0),
	}
}

func (m *matchType) Name() string {
	return fmt.Sprintf("Expect type: %s", m.typ.Type().String())
}

func (m *matchType) Met(ctx Context) (unmet error, err error) {
	ov := OperandValue{Original: m.value}
	if ov.Resolved, err = ResolveValue(ov.Original, ctx); err == nil {
		if ov.Resolved != nil {
			to := reflect.TypeOf(ov.Resolved)
			if ok := m.typ.Type() == to; !ok {
				ov.Coerced = to.String()
				unmet = &unmetError{
					msg:      fmt.Sprintf("expected type %q", m.typ.Type().String()),
					name:     m.Name(),
					expected: OperandValue{Original: m.typ.Type().String()},
					actual:   ov,
					frame:    m.frame,
				}
			}
		} else {
			unmet = &unmetError{
				msg:      "expected type on nil",
				name:     m.Name(),
				expected: OperandValue{Original: m.typ.Type().String()},
				actual:   ov,
				frame:    m.frame,
			}
		}
	}
	return
}

func (m *matchType) Frame() *framing.Frame {
	return m.frame
}

type nilCheck struct {
	value any
	frame *framing.Frame
	commonExpectation
}

var _ Expectation = (*nilCheck)(nil)

// ExpectNil asserts that the value (or resolved value) is nil
//
//go:noinline
func ExpectNil(value any) Expectation {
	return &nilCheck{
		value: value,
		frame: framing.NewFrame(0),
	}
}

func (n *nilCheck) Name() string {
	return "Expect Nil"
}

func (n *nilCheck) Met(ctx Context) (unmet error, err error) {
	ov := OperandValue{Original: n.value}
	if ov.Resolved, err = ResolveValue(ov.Original, ctx); err == nil {
		if ok := ov.Resolved == nil; !ok {
			unmet = &unmetError{
				msg:    fmt.Sprintf("expected nil"),
				name:   n.Name(),
				actual: ov,
				frame:  n.frame,
			}
		}
	}
	return
}

func (n *nilCheck) Frame() *framing.Frame {
	return n.frame
}

type notNilCheck struct {
	value any
	frame *framing.Frame
	commonExpectation
}

var _ Expectation = (*notNilCheck)(nil)

// ExpectNotNil asserts that the value (or resolved value) is not nil
//
//go:noinline
func ExpectNotNil(value any) Expectation {
	return &notNilCheck{
		value: value,
		frame: framing.NewFrame(0),
	}
}

func (n *notNilCheck) Name() string {
	return "Expect Not Nil"
}

func (n *notNilCheck) Met(ctx Context) (unmet error, err error) {
	ov := OperandValue{Original: n.value}
	if ov.Resolved, err = ResolveValue(ov.Original, ctx); err == nil {
		if ok := ov.Resolved != nil; !ok {
			unmet = &unmetError{
				msg:    fmt.Sprintf("expected not nil"),
				name:   n.Name(),
				actual: ov,
				frame:  n.frame,
			}
		}
	}
	return
}

func (n *notNilCheck) Frame() *framing.Frame {
	return n.frame
}

type lenCheck struct {
	value  any
	length int
	frame  *framing.Frame
	commonExpectation
}

var _ Expectation = (*lenCheck)(nil)

// ExpectLen asserts that the value (or resolved value) has the supplied length
//
// the value (or resolved value) must be a string, map or slice
//
//go:noinline
func ExpectLen(value any, length int) Expectation {
	return &lenCheck{
		value:  value,
		length: length,
		frame:  framing.NewFrame(0),
	}
}

func (l *lenCheck) Name() string {
	return "Expect Len"
}

func (l *lenCheck) Met(ctx Context) (unmet error, err error) {
	ov := OperandValue{Original: l.value}
	if ov.Resolved, err = ResolveValue(ov.Original, ctx); err == nil {
		ok := false
		switch avt := ov.Resolved.(type) {
		case string:
			ov.Resolved = len(avt)
			ok = len(avt) == l.length
		case map[string]any:
			ov.Resolved = len(avt)
			ok = len(avt) == l.length
		case []any:
			ov.Resolved = len(avt)
			ok = len(avt) == l.length
		default:
			checked := false
			if ov.Resolved != nil {
				to := reflect.ValueOf(ov.Resolved)
				if to.Kind() == reflect.Map || to.Kind() == reflect.Slice {
					checked = true
					ov.Resolved = to.Len()
					ok = to.Len() == l.length
				}
			}
			if !checked {
				unmet = &unmetError{
					msg:      fmt.Sprintf("cannot check length on %T", ov.Resolved),
					name:     l.Name(),
					expected: OperandValue{Original: l.length, Resolved: l.length},
					actual:   ov,
					frame:    l.frame,
				}
				return
			}
		}
		if !ok {
			unmet = &unmetError{
				msg:      fmt.Sprintf("expected length %d", l.length),
				name:     l.Name(),
				expected: OperandValue{Original: l.length, Resolved: l.length},
				actual:   ov,
				frame:    l.frame,
			}
		}
	}
	return
}

func (l *lenCheck) Frame() *framing.Frame {
	return l.frame
}

type expectMockCall struct {
	name   string
	path   string
	method string
	frame  *framing.Frame
	commonExpectation
}

var _ Expectation = (*expectMockCall)(nil)

// ExpectMockServiceCalled asserts that a specific mock service endpoint+method was called
//
//go:noinline
func ExpectMockServiceCalled(svcName string, path string, method MethodName) Expectation {
	return &expectMockCall{
		name:   svcName,
		path:   path,
		method: strings.ToUpper(string(method)),
		frame:  framing.NewFrame(0),
	}
}

func (e *expectMockCall) Name() string {
	return "EXPECT MOCK SERVICE CALL [" + e.name + "]: " + e.method + " " + e.path
}

func (e *expectMockCall) Frame() *framing.Frame {
	return e.frame
}

func (e *expectMockCall) Met(ctx Context) (unmet error, err error) {
	if ms := ctx.GetMockService(e.name); ms != nil {
		var actualPath string
		if actualPath, err = resolveValueString(e.path, ctx); err == nil {
			if !ms.AssertCalled(actualPath, e.method) {
				unmet = &unmetError{
					msg:      fmt.Sprintf("expected mock service call [%s]: %s %s", e.name, e.method, actualPath),
					name:     e.Name(),
					expected: OperandValue{Original: true, Resolved: true},
					actual:   OperandValue{Original: false, Resolved: false},
					frame:    e.frame,
				}
			}
		}
		return
	}
	return nil, fmt.Errorf("unknown mock service %q", e.name)
}

type propertiesCheck struct {
	value      any
	properties []string
	only       bool
	frame      *framing.Frame
	commonExpectation
}

var _ Expectation = (*propertiesCheck)(nil)

// ExpectHasProperties asserts that the value (or resolved value) has the supplied properties
//
// the value (or resolved value) must be a map
//
//go:noinline
func ExpectHasProperties(value any, propertyNames ...string) Expectation {
	return &propertiesCheck{
		value:      value,
		properties: propertyNames,
		frame:      framing.NewFrame(0),
	}
}

// ExpectOnlyHasProperties asserts that the value (or resolved value) only has the supplied properties
//
// the value (or resolved value) must be a map
//
//go:noinline
func ExpectOnlyHasProperties(value any, propertyNames ...string) Expectation {
	return &propertiesCheck{
		value:      value,
		properties: propertyNames,
		only:       true,
		frame:      framing.NewFrame(0),
	}
}

func (p *propertiesCheck) Name() string {
	if p.only {
		return "Expect Only Properties"
	}
	return "Expect Properties"
}

func (p *propertiesCheck) Frame() *framing.Frame {
	return p.frame
}

func (p *propertiesCheck) Met(ctx Context) (unmet error, err error) {
	var av any
	if av, err = ResolveValue(p.value, ctx); err == nil {
		checked := false
		keys := make(map[string]struct{}, len(p.properties))
		kNames := make([]string, 0, len(p.properties))
		switch avt := av.(type) {
		case map[string]any:
			checked = true
			for k := range avt {
				keys[k] = struct{}{}
				kNames = append(kNames, k)
			}
		default:
			if av != nil {
				to := reflect.ValueOf(av)
				if to.Kind() == reflect.Map && to.Type().Key().Kind() == reflect.String {
					checked = true
					iter := to.MapRange()
					for iter.Next() {
						keys[iter.Key().Interface().(string)] = struct{}{}
						kNames = append(kNames, iter.Key().Interface().(string))
					}
				}
			}
		}
		if !checked {
			sort.Strings(p.properties)
			expectStr := `"` + strings.Join(p.properties, `", "`) + `"`
			unmet = &unmetError{
				msg:      fmt.Sprintf("cannot check properties on %T", av),
				name:     p.Name(),
				expected: OperandValue{Original: expectStr, Resolved: expectStr},
				actual:   OperandValue{Original: p.value, Resolved: av},
				frame:    p.frame,
			}
		} else {
			ok := true
			for _, prop := range p.properties {
				if _, ok = keys[prop]; !ok {
					ok = false
					break
				} else {
					delete(keys, prop)
				}
			}
			if !ok || (ok && p.only && len(keys) > 0) {
				sort.Strings(p.properties)
				expectStr := `"` + strings.Join(p.properties, `", "`) + `"`
				sort.Strings(kNames)
				actualStr := `"` + strings.Join(kNames, `", "`) + `"`
				unmet = &unmetError{
					msg:      "expected properties",
					name:     p.Name(),
					expected: OperandValue{Original: expectStr, Resolved: expectStr},
					actual:   OperandValue{Original: actualStr, Resolved: actualStr},
					frame:    p.frame,
				}
			}
		}
	}
	return
}

type failCheck struct {
	msg   string
	frame *framing.Frame
	commonExpectation
}

var _ Expectation = (*failCheck)(nil)

// Fail is an expectation that always fails (not just unmet)
//
// can be used in Method.If to cause test fail conditions
//
//go:noinline
func Fail(msg string) Expectation {
	return &failCheck{
		msg:   msg,
		frame: framing.NewFrame(0),
	}
}

func (f *failCheck) Name() string {
	return fmt.Sprintf("FAIL %q", f.msg)
}

func (f *failCheck) Frame() *framing.Frame {
	return f.frame
}

func (f *failCheck) Met(ctx Context) (unmet error, err error) {
	return nil, errors.New(f.msg)
}

type varCheck struct {
	name  Var
	frame *framing.Frame
	commonExpectation
}

var _ Expectation = (*varCheck)(nil)

// ExpectVarSet asserts that a named variable has been set
//
//go:noinline
func ExpectVarSet(v Var) Expectation {
	return &varCheck{
		name:  v,
		frame: framing.NewFrame(0),
	}
}

func (v varCheck) Name() string {
	return fmt.Sprintf("Expect Var(%q) set", string(v.name))
}

func (v varCheck) Frame() *framing.Frame {
	return v.frame
}

func (v varCheck) Met(ctx Context) (unmet error, err error) {
	if _, ok := ctx.Vars()[v.name]; !ok {
		unmet = &unmetError{
			msg:      "expected variable set",
			name:     v.Name(),
			expected: OperandValue{Original: v.name, Resolved: true},
			actual:   OperandValue{Original: false, Resolved: false},
			frame:    v.frame,
		}
	}
	return unmet, nil
}
