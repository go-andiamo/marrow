package marrow

import (
	"errors"
	"github.com/go-andiamo/marrow/framing"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestExpectationFunc(t *testing.T) {
	t.Run("met", func(t *testing.T) {
		exp := ExpectationFunc(func(ctx Context) (unmet error, err error) {
			return nil, nil
		})
		assert.Equal(t, "(User Defined Expectation)", exp.Name())
		assert.NotNil(t, exp.Frame())
		unmet, err := exp.Met(nil)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet", func(t *testing.T) {
		exp := ExpectationFunc(func(ctx Context) (unmet error, err error) {
			return errors.New("fooey"), nil
		})
		assert.Equal(t, "(User Defined Expectation)", exp.Name())
		unmet, err := exp.Met(nil)
		assert.Error(t, unmet)
		assert.NoError(t, err)
		umerr, ok := unmet.(UnmetError)
		assert.True(t, ok)
		assert.Error(t, umerr)
		assert.Equal(t, "user defined expectation failed", umerr.Error())
		err = errors.Unwrap(umerr)
		assert.Error(t, err)
		assert.Equal(t, "fooey", err.Error())
	})

}

func Test_expectStatusCode(t *testing.T) {
	exp := &expectStatusCode{
		name:   "Expect Status Code",
		expect: http.StatusConflict,
		frame:  framing.NewFrame(0),
	}
	assert.Equal(t, "Expect Status Code", exp.Name())
	assert.NotNil(t, exp.Frame())
	assert.False(t, exp.IsRequired())
	t.Run("met", func(t *testing.T) {
		ctx := newTestContext(nil)
		ctx.currResponse = &http.Response{StatusCode: http.StatusConflict}
		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet", func(t *testing.T) {
		ctx := newTestContext(nil)
		ctx.currResponse = &http.Response{StatusCode: http.StatusNotFound}
		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
		umerr, ok := unmet.(UnmetError)
		assert.True(t, ok)
		assert.Error(t, umerr)
		assert.Equal(t, http.StatusConflict, umerr.Expected().Original)
		assert.Equal(t, http.StatusNotFound, umerr.Actual().Original)
	})
	t.Run("missing var", func(t *testing.T) {
		ctx := newTestContext(nil)
		ctx.currResponse = &http.Response{StatusCode: http.StatusNotFound}
		exp := &expectStatusCode{
			name:   "Expect Status Code",
			expect: Var("missing"),
			frame:  framing.NewFrame(0),
		}
		_, err := exp.Met(ctx)
		assert.Error(t, err)
	})
	t.Run("expect string", func(t *testing.T) {
		ctx := newTestContext(nil)
		ctx.currResponse = &http.Response{StatusCode: http.StatusNotFound}
		exp := &expectStatusCode{
			name:   "Expect Status Code",
			expect: "404",
			frame:  framing.NewFrame(0),
		}
		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("expect int64", func(t *testing.T) {
		ctx := newTestContext(nil)
		ctx.currResponse = &http.Response{StatusCode: http.StatusNotFound}
		exp := &expectStatusCode{
			name:   "Expect Status Code",
			expect: int64(http.StatusNotFound),
			frame:  framing.NewFrame(0),
		}
		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("invalid type", func(t *testing.T) {
		ctx := newTestContext(nil)
		ctx.currResponse = &http.Response{StatusCode: http.StatusNotFound}
		exp := &expectStatusCode{
			name:   "Expect Status Code",
			expect: true,
			frame:  framing.NewFrame(0),
		}
		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
}

func TestStatus_stringify(t *testing.T) {
	sc := Status(http.StatusNotFound)
	assert.Equal(t, `404 "Not Found"`, sc.stringify())
	sc = Status(999)
	assert.Equal(t, `999`, sc.stringify())
}

func Test_match(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		exp := &match{
			value: "foo",
			regex: "[a-z]{3}",
			frame: framing.NewFrame(0),
		}
		assert.Equal(t, "Expect match: \"[a-z]{3}\"", exp.Name())
		assert.NotNil(t, exp.Frame())
	})
	t.Run("met", func(t *testing.T) {
		exp := &match{
			value: "foo",
			regex: "[a-z]{3}",
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("met (map)", func(t *testing.T) {
		exp := &match{
			value: map[string]any{"foo": nil},
			regex: "[a-z]{3}",
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("bad regex", func(t *testing.T) {
		exp := &match{
			regex: "[",
			frame: framing.NewFrame(0),
		}
		_, err := exp.Met(nil)
		assert.Error(t, err)
	})
	t.Run("missing var", func(t *testing.T) {
		exp := &match{
			value: Var("foo"),
			regex: "[a-z]{3}",
			frame: framing.NewFrame(0),
		}
		_, err := exp.Met(newTestContext(nil))
		assert.Error(t, err)
	})
	t.Run("unmet (int var)", func(t *testing.T) {
		exp := &match{
			value: Var("foo"),
			regex: "[a-z]{3}",
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(newTestContext(map[Var]any{
			"foo": 42,
		}))
		assert.Error(t, unmet)
		assert.NoError(t, err)
		umerr, ok := unmet.(UnmetError)
		assert.True(t, ok)
		assert.Error(t, umerr)
		assert.Equal(t, Var("foo"), umerr.Actual().Original)
		assert.Equal(t, 42, umerr.Actual().Resolved)
		assert.Equal(t, "42", umerr.Actual().Coerced)
	})
	t.Run("unmet (marshal error)", func(t *testing.T) {
		exp := &match{
			value: map[string]any{"x": &unmarshalable{}},
			regex: "[a-z]{3}",
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.Error(t, unmet)
		assert.NoError(t, err)
		umerr, ok := unmet.(UnmetError)
		assert.True(t, ok)
		assert.Error(t, umerr)
		assert.Error(t, umerr.Actual().CoercionError)
	})
}

func Test_lenCheck(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		exp := &lenCheck{
			value:  "foo",
			length: 3,
			frame:  framing.NewFrame(0),
		}
		assert.Equal(t, "Expect Len", exp.Name())
		assert.NotNil(t, exp.Frame())
	})
	t.Run("met (string)", func(t *testing.T) {
		exp := &lenCheck{
			value:  "foo",
			length: 3,
			frame:  framing.NewFrame(0),
		}
		unmet, err := exp.Met(newTestContext(nil))
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("met (map[string]any)", func(t *testing.T) {
		exp := &lenCheck{
			value:  map[string]any{"foo": nil, "bar": nil, "baz": nil},
			length: 3,
			frame:  framing.NewFrame(0),
		}
		unmet, err := exp.Met(newTestContext(nil))
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("met ([]any)", func(t *testing.T) {
		exp := &lenCheck{
			value:  []any{nil, nil, nil},
			length: 3,
			frame:  framing.NewFrame(0),
		}
		unmet, err := exp.Met(newTestContext(nil))
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("met (map)", func(t *testing.T) {
		exp := &lenCheck{
			value:  map[string]string{"foo": "", "bar": "", "baz": ""},
			length: 3,
			frame:  framing.NewFrame(0),
		}
		unmet, err := exp.Met(newTestContext(nil))
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("met (slice)", func(t *testing.T) {
		exp := &lenCheck{
			value:  []string{"", "", ""},
			length: 3,
			frame:  framing.NewFrame(0),
		}
		unmet, err := exp.Met(newTestContext(nil))
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet", func(t *testing.T) {
		exp := &lenCheck{
			value:  "foo",
			length: 4,
			frame:  framing.NewFrame(0),
		}
		unmet, err := exp.Met(newTestContext(nil))
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet (invalid type)", func(t *testing.T) {
		exp := &lenCheck{
			value:  true,
			length: 4,
			frame:  framing.NewFrame(0),
		}
		unmet, err := exp.Met(newTestContext(nil))
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("missing var", func(t *testing.T) {
		exp := &lenCheck{
			value:  Var("foo"),
			length: 3,
			frame:  framing.NewFrame(0),
		}
		_, err := exp.Met(newTestContext(nil))
		assert.Error(t, err)
	})
}

type unmarshalable struct{}

func (*unmarshalable) MarshalJSON() ([]byte, error) {
	return nil, errors.New("fooey")
}

func Test_matchType(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		exp := &matchType{
			value: "foo",
			typ:   Type[string](),
			frame: framing.NewFrame(0),
		}
		assert.Equal(t, "Expect type: string", exp.Name())
		assert.NotNil(t, exp.Frame())
	})
	t.Run("met", func(t *testing.T) {
		exp := &matchType{
			value: "foo",
			typ:   Type[string](),
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet", func(t *testing.T) {
		exp := &matchType{
			value: 42,
			typ:   Type[string](),
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.Error(t, unmet)
		assert.NoError(t, err)
		umerr, ok := unmet.(UnmetError)
		assert.True(t, ok)
		assert.Error(t, umerr)
		assert.Equal(t, "string", umerr.Expected().Original)
		assert.Equal(t, 42, umerr.Actual().Original)
		assert.Equal(t, 42, umerr.Actual().Resolved)
		assert.Equal(t, "int", umerr.Actual().Coerced)
	})
	t.Run("unmet on nil", func(t *testing.T) {
		exp := &matchType{
			value: nil,
			typ:   Type[string](),
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.Error(t, unmet)
		assert.NoError(t, err)
		umerr, ok := unmet.(UnmetError)
		assert.True(t, ok)
		assert.Error(t, umerr)
		assert.Equal(t, "string", umerr.Expected().Original)
		assert.Nil(t, umerr.Actual().Original)
		assert.Nil(t, umerr.Actual().Resolved)
	})
	t.Run("missing var", func(t *testing.T) {
		exp := &matchType{
			value: Var("foo"),
			typ:   Type[string](),
			frame: framing.NewFrame(0),
		}
		_, err := exp.Met(newTestContext(nil))
		assert.Error(t, err)
	})
}

func Test_nilCheck(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		exp := &nilCheck{
			value: "foo",
			frame: framing.NewFrame(0),
		}
		assert.Equal(t, "Expect Nil", exp.Name())
		assert.NotNil(t, exp.Frame())
	})
	t.Run("met", func(t *testing.T) {
		exp := &nilCheck{
			value: nil,
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet", func(t *testing.T) {
		exp := &nilCheck{
			value: "not nil",
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.Error(t, unmet)
		assert.NoError(t, err)
		umerr, ok := unmet.(UnmetError)
		assert.True(t, ok)
		assert.Error(t, umerr)
		assert.Nil(t, umerr.Expected().Original)
		assert.Equal(t, "not nil", umerr.Actual().Original)
		assert.Equal(t, "not nil", umerr.Actual().Resolved)
	})
	t.Run("missing var", func(t *testing.T) {
		exp := &nilCheck{
			value: Var("foo"),
			frame: framing.NewFrame(0),
		}
		_, err := exp.Met(newTestContext(nil))
		assert.Error(t, err)
	})
}

func Test_notNilCheck(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		exp := &notNilCheck{
			value: "foo",
			frame: framing.NewFrame(0),
		}
		assert.Equal(t, "Expect Not Nil", exp.Name())
		assert.NotNil(t, exp.Frame())
	})
	t.Run("met", func(t *testing.T) {
		exp := &notNilCheck{
			value: "not nil",
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet", func(t *testing.T) {
		exp := &notNilCheck{
			value: nil,
			frame: framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.Error(t, unmet)
		assert.NoError(t, err)
		umerr, ok := unmet.(UnmetError)
		assert.True(t, ok)
		assert.Error(t, umerr)
		assert.Nil(t, umerr.Expected().Original)
		assert.Nil(t, umerr.Actual().Original)
		assert.Nil(t, umerr.Actual().Resolved)
	})
	t.Run("missing var", func(t *testing.T) {
		exp := &notNilCheck{
			value: Var("foo"),
			frame: framing.NewFrame(0),
		}
		_, err := exp.Met(newTestContext(nil))
		assert.Error(t, err)
	})
}
