package marrow

import (
	"fmt"
	"github.com/go-andiamo/marrow/framing"
	"reflect"
	"strconv"
	"strings"
)

type ErrorType int

const (
	ErrorUnmet ErrorType = iota
	ErrorCapture
)

// Error represents the basic error for almost all errors produced
type Error interface {
	error
	Type() ErrorType
	Name() string
	Cause() error
	Unwrap() error
	TestFormat() string
	framing.Framed
}

// UnmetError represents specific information about an expectation (assert/require) that was unmet
type UnmetError interface {
	Error
	Expected() OperandValue
	Actual() OperandValue
	IsComparator() bool
	Comparator() string
	Left() OperandValue
	Right() OperandValue
}

// OperandValue is a representation of an operand value - giving the original, resolved & coerced value
type OperandValue struct {
	Original      any
	Resolved      any
	Coerced       any
	CoercionError error
}

type stringy interface {
	stringify() string
}

type unmetError struct {
	msg          string
	name         string
	expected     OperandValue
	actual       OperandValue
	isComparator bool
	comparator   string
	left         OperandValue
	right        OperandValue
	cause        error
	frame        *framing.Frame
}

var _ UnmetError = (*unmetError)(nil)

func (e *unmetError) Error() string {
	return e.msg
}

func (e *unmetError) Type() ErrorType {
	return ErrorUnmet
}

func (e *unmetError) Name() string {
	return e.name
}

func (e *unmetError) Expected() OperandValue {
	return e.expected
}

func (e *unmetError) Actual() OperandValue {
	return e.actual
}

func (e *unmetError) IsComparator() bool {
	return e.isComparator
}

func (e *unmetError) Comparator() string {
	return e.comparator
}

func (e *unmetError) Left() OperandValue {
	return e.left
}

func (e *unmetError) Right() OperandValue {
	return e.right
}

func (e *unmetError) Cause() error {
	return e.cause
}

func (e *unmetError) Unwrap() error {
	return e.cause
}

func (e *unmetError) Frame() *framing.Frame {
	return e.frame
}

func (e *unmetError) TestFormat() string {
	var b strings.Builder
	b.WriteString(e.msg)
	if e.isComparator {
		b.WriteString(fmt.Sprintf("\n\tLeft:     \t%v", e.left.TestFormat()))
		b.WriteString(fmt.Sprintf("\n\tRight:    \t%v", e.right.TestFormat()))
	} else {
		b.WriteString(fmt.Sprintf("\n\tExpected: \t%v", e.expected.TestFormat()))
		b.WriteString(fmt.Sprintf("\n\tActual:   \t%v", e.actual.TestFormat()))
	}
	if e.cause != nil {
		b.WriteString(fmt.Sprintf("\n\tCause:    \t%s", e.cause.Error()))
	}
	if e.frame != nil {
		b.WriteString(fmt.Sprintf("\n\tFrame:    \t%s:%d", e.frame.File, e.frame.Line))
	}
	return b.String()
}

type captureError struct {
	msg     string
	name    string
	cause   error
	frame   *framing.Frame
	capture Capture
	values  []OperandValue
}

type CaptureError interface {
	Error
	Capture() Capture
	Values() []OperandValue
}

func wrapCaptureError(cause error, msg string, capture Capture, values ...OperandValue) error {
	if cause == nil {
		return nil
	}
	if msg == "" {
		msg = cause.Error()
	}
	return &captureError{
		msg:     msg,
		name:    capture.Name(),
		cause:   cause,
		frame:   capture.Frame(),
		capture: capture,
		values:  values,
	}
}

func newCaptureError(msg string, cause error, capture Capture, values ...OperandValue) error {
	return &captureError{
		msg:     msg,
		name:    capture.Name(),
		cause:   cause,
		frame:   capture.Frame(),
		capture: capture,
		values:  values,
	}
}

var _ Error = (*captureError)(nil)
var _ CaptureError = (*captureError)(nil)

func (e *captureError) Capture() Capture {
	return e.capture
}

func (e *captureError) Values() []OperandValue {
	return e.values
}

func (e *captureError) Error() string {
	return e.msg
}

func (e *captureError) Type() ErrorType {
	return ErrorCapture
}

func (e *captureError) Name() string {
	return e.name
}

func (e *captureError) Cause() error {
	return e.cause
}

func (e *captureError) Unwrap() error {
	return e.cause
}

func (e *captureError) Frame() *framing.Frame {
	return e.frame
}

func (e *captureError) TestFormat() string {
	var b strings.Builder
	b.WriteString(e.msg)
	for _, ov := range e.values {
		out := false
		if ov.Original != nil {
			if _, ok := ov.Original.(Resolvable); ok {
				if sv, ok := ov.Original.(fmt.Stringer); ok {
					out = true
					b.WriteString(fmt.Sprintf("\n\tValue:    \t%s", sv.String()))
				}
			}
		}
		if !out {
			t := trimPackagePrefix(fmt.Sprintf("%T", ov.Original))
			b.WriteString(fmt.Sprintf("\n\tValue:    \t%s(%v)", t, ov.Original))
		}
	}
	if e.cause != nil {
		b.WriteString(fmt.Sprintf("\n\tCause:    \t%s", e.cause.Error()))
	}
	if e.frame != nil {
		b.WriteString(fmt.Sprintf("\n\tFrame:    \t%s:%d", e.frame.File, e.frame.Line))
	}
	return b.String()
}

func (v OperandValue) TestFormat() string {
	var b strings.Builder
	switch rt := v.Resolved.(type) {
	case nil:
		b.WriteString(fmt.Sprintf("%v", v.Original))
	case stringy:
		b.WriteString(rt.stringify())
	case string:
		b.WriteString(strconv.Quote(rt))
	default:
		to := reflect.TypeOf(v.Resolved)
		b.WriteString(fmt.Sprintf("%s(%v)", to.String(), rt))
	}
	switch ot := v.Original.(type) {
	case fmt.Stringer:
		b.WriteString(fmt.Sprintf(" << %s", ot.String()))
	case Resolvable:
		to := reflect.TypeOf(v.Original)
		b.WriteString(fmt.Sprintf(" << %s(%v)", trimPackagePrefix(to.String()), ot))
	}
	if v.CoercionError != nil {
		b.WriteString(fmt.Sprintf("\n\t          \tCoercion error: %s", v.CoercionError.Error()))
	}
	return b.String()
}

func trimPackagePrefix(s string) string {
	return strings.TrimPrefix(strings.TrimPrefix(s, "marrow."), "*marrow.")
}
