package marrow

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
)

type Expectation interface {
	Name() string
	Met(ctx Context) (unmet error, err error)
	Framed
}

//go:noinline
func ExpectationFunc(fn func(ctx Context) (unmet error, err error)) Expectation {
	return &expectation{
		fn:    fn,
		frame: frame(0),
	}
}

type expectation struct {
	fn    func(ctx Context) (unmet error, err error)
	frame *Frame
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

func (e *expectation) Frame() *Frame {
	return e.frame
}

/*
//go:noinline
func newExpectOk(skip int) Expectation {
	return &expectStatusCode{
		name:   "Expect OK",
		frame:  frame(skip),
		expect: http.StatusOK,
	}
}

//go:noinline
func newExpectStatusCode(skip int, status any) Expectation {
	return &expectStatusCode{
		name:   "Expect Status Code",
		frame:  frame(skip),
		expect: status,
	}
}
*/

type expectStatusCode struct {
	name   string
	expect any
	frame  *Frame
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
	case float64:
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

func (e *expectStatusCode) Frame() *Frame {
	return e.frame
}

type match struct {
	value any
	regex string
	rx    *regexp.Regexp
	frame *Frame
}

var _ Expectation = (*match)(nil)

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
			if to.Kind() == reflect.Map || to.Kind() == reflect.Slice {
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

func (m *match) Frame() *Frame {
	return m.frame
}

type matchType struct {
	value any
	typ   Type_
	frame *Frame
}

var _ Expectation = (*matchType)(nil)

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

func (m *matchType) Frame() *Frame {
	return m.frame
}

type nilCheck struct {
	value any
	frame *Frame
}

var _ Expectation = (*nilCheck)(nil)

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

func (n *nilCheck) Frame() *Frame {
	return n.frame
}

type notNilCheck struct {
	value any
	frame *Frame
}

var _ Expectation = (*notNilCheck)(nil)

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

func (n *notNilCheck) Frame() *Frame {
	return n.frame
}

type lenCheck struct {
	value  any
	length int
	frame  *Frame
}

var _ Expectation = (*lenCheck)(nil)

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

func (l *lenCheck) Frame() *Frame {
	return l.frame
}
