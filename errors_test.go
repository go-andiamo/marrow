package marrow

import (
	"errors"
	"github.com/go-andiamo/marrow/framing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestUnmetError(t *testing.T) {
	err := &unmetError{
		msg:          "expected something",
		name:         "expected ==",
		isComparator: true,
		comparator:   "==",
		left:         OperandValue{},
		right:        OperandValue{},
		cause:        errors.New("fooey"),
		frame:        framing.NewFrame(0),
	}
	assert.Error(t, err)
	assert.Equal(t, "expected something", err.Error())
	var cet UnmetError
	assert.True(t, errors.As(err, &cet))
	var umerr UnmetError = err
	assert.ErrorIs(t, umerr, cet)
	assert.Equal(t, ErrorUnmet, umerr.Type())
	assert.Equal(t, "expected ==", umerr.Name())
	assert.True(t, umerr.IsComparator())
	assert.Equal(t, "==", umerr.Comparator())
	assert.Error(t, umerr.Cause())
	assert.Error(t, umerr.Unwrap())
	assert.NotNil(t, umerr.Frame())
	assert.NotNil(t, umerr.Left())
	assert.NotNil(t, umerr.Right())
	assert.NotNil(t, umerr.Expected())
	assert.NotNil(t, umerr.Actual())
}

func TestUnmetError_Comparator_TestFormat(t *testing.T) {
	err := &unmetError{
		msg:          "expected something",
		name:         "expected ==",
		isComparator: true,
		comparator:   "==",
		left:         OperandValue{Original: Var("xxx"), Resolved: "xxx"},
		right:        OperandValue{Original: Var("yyy"), Resolved: "yyy"},
		cause:        errors.New("fooey"),
		frame:        framing.NewFrame(0),
	}
	assert.Error(t, err)
	var cet UnmetError
	assert.True(t, errors.As(err, &cet))
	var umerr UnmetError = err
	assert.ErrorIs(t, umerr, cet)
	s := umerr.TestFormat()
	assert.Contains(t, s, "expected something\n")
	assert.Contains(t, s, "\tLeft:     \t\"xxx\" << Var(xxx)\n")
	assert.Contains(t, s, "\tRight:    \t\"yyy\" << Var(yyy)\n")
	assert.Contains(t, s, "\tFrame:    \t")
}

func TestUnmetError_NonComparator_TestFormat(t *testing.T) {
	err := &unmetError{
		msg:          "expected something",
		name:         "expected ==",
		isComparator: false,
		comparator:   "==",
		expected:     OperandValue{Original: Var("xxx"), Resolved: "xxx"},
		actual:       OperandValue{Original: Var("yyy"), Resolved: "yyy"},
		cause:        errors.New("fooey"),
		frame:        framing.NewFrame(0),
	}
	assert.Error(t, err)
	var cet UnmetError
	assert.True(t, errors.As(err, &cet))
	var umerr UnmetError = err
	assert.ErrorIs(t, umerr, cet)
	s := umerr.TestFormat()
	assert.Contains(t, s, "expected something\n")
	assert.Contains(t, s, "\tExpected: \t\"xxx\" << Var(xxx)\n")
	assert.Contains(t, s, "\tActual:   \t\"yyy\" << Var(yyy)\n")
	assert.Contains(t, s, "\tFrame:    \t")
}

func Test_newCaptureError(t *testing.T) {
	c := &setVar{
		name:  "foo",
		value: "bar",
		frame: framing.NewFrame(0),
	}
	err := newCaptureError("whoops", errors.New("fooey"), c)
	require.Error(t, err)
	var cet CaptureError
	assert.True(t, errors.As(err, &cet))
	ce, ok := cet.(CaptureError)
	assert.True(t, ok)
	assert.ErrorIs(t, ce, cet)
	assert.Equal(t, ErrorCapture, ce.Type())
	assert.Equal(t, "whoops", ce.Error())
	assert.Equal(t, "foo", ce.Name())
	assert.Equal(t, "foo", ce.Capture().Name())
	assert.Empty(t, ce.Values())
	assert.Error(t, ce.Cause())
	assert.Error(t, ce.Unwrap())
	assert.NotNil(t, ce.Frame())
}

func TestCaptureError_TestFormat(t *testing.T) {
	c := &setVar{
		name:  "foo",
		value: "bar",
		frame: framing.NewFrame(0),
	}
	ov1 := OperandValue{
		Original: Var("foo"),
	}
	ov2 := OperandValue{
		Original: Body,
	}
	ov3 := OperandValue{
		Original: "xxx",
	}
	err := newCaptureError("whoops", errors.New("fooey"), c, ov1, ov2, ov3)
	require.Error(t, err)
	ce, ok := err.(CaptureError)
	assert.True(t, ok)
	s := ce.TestFormat()
	assert.Contains(t, s, "whoops\n")
	assert.Contains(t, s, "\tValue:    \tVar(foo)\n")
	assert.Contains(t, s, "\tValue:    \tBody\n")
	assert.Contains(t, s, "\tValue:    \tstring(xxx)\n")
	assert.Contains(t, s, "\tCause:    \tfooey\n")
	assert.Contains(t, s, "\tFrame:    \t")
}

func Test_wrapCaptureError(t *testing.T) {
	c := &setVar{
		name:  "foo",
		value: "bar",
		frame: framing.NewFrame(0),
	}
	err := wrapCaptureError(nil, "whoops", c)
	require.NoError(t, err)

	err = wrapCaptureError(errors.New("fooey"), "", c)
	require.Error(t, err)
	assert.Equal(t, "fooey", err.Error())
}

func TestOperandValue_TestFormat(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		ov := OperandValue{
			Original: nil,
			Resolved: nil,
		}
		s := ov.TestFormat()
		assert.Contains(t, s, "<nil>")
	})
	t.Run("var", func(t *testing.T) {
		ov := OperandValue{
			Original: Var("foo"),
			Resolved: "bar",
		}
		s := ov.TestFormat()
		assert.Contains(t, s, "\"bar\" << Var(foo)")
	})
	t.Run("resolved", func(t *testing.T) {
		ov := OperandValue{
			Original: Var("foo"),
			Resolved: "bar",
		}
		s := ov.TestFormat()
		assert.Contains(t, s, "\"bar\" << Var(foo)")
	})
	t.Run("resolved, stringify", func(t *testing.T) {
		ov := OperandValue{
			Original: Var("foo"),
			Resolved: Var("bar"),
		}
		s := ov.TestFormat()
		assert.Contains(t, s, "marrow.Var(Var(bar)) << Var(foo)")
	})
	t.Run("stringify", func(t *testing.T) {
		ov := OperandValue{
			Original: 404,
			Resolved: Status(http.StatusNotFound),
		}
		s := ov.TestFormat()
		assert.Contains(t, s, "404 \"Not Found\"")
	})
	t.Run("misc type", func(t *testing.T) {
		ov := OperandValue{
			Original: Var("foo"),
			Resolved: struct{}{},
		}
		s := ov.TestFormat()
		assert.Contains(t, s, "struct {}({}) << Var(foo)")
	})
	t.Run("coercion error", func(t *testing.T) {
		ov := OperandValue{
			Original:      Var("foo"),
			Resolved:      "bar",
			CoercionError: errors.New("fooey"),
		}
		s := ov.TestFormat()
		assert.Contains(t, s, "\"bar\" << Var(foo)")
		assert.Contains(t, s, "\tCoercion error: fooey")
	})
}
