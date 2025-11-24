package marrow

import (
	"errors"
	"fmt"
	"github.com/go-andiamo/marrow/framing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	t.Run("panic on direct .Run()", func(t *testing.T) {
		exp := &expectStatusCode{
			name:   "Expect Status Code",
			expect: true,
			frame:  framing.NewFrame(0),
		}
		require.Panics(t, func() {
			_ = exp.Run(newTestContext(nil))
		})
	})
}

func TestStatus_stringify(t *testing.T) {
	sc := Status(http.StatusNotFound)
	assert.Equal(t, `404 "Not Found"`, sc.stringify())
	sc = Status(999)
	assert.Equal(t, `999`, sc.stringify())
}

func Test_expectStatusCodeIn(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		exp := ExpectStatusCodeIn(http.StatusNotFound, http.StatusOK)
		assert.Equal(t, `Expect Status Code in (404 "Not Found", 200 "OK")`, exp.Name())
		assert.NotNil(t, exp.Frame())
		assert.False(t, exp.IsRequired())
	})
	t.Run("met", func(t *testing.T) {
		exp := ExpectStatusCodeIn(nil, "404", int64(404), Var("foo"), http.StatusNotFound, http.StatusOK)
		assert.Equal(t, `Expect Status Code in ("404", 404 "Not Found", Var(foo), 404 "Not Found", 200 "OK")`, exp.Name())
		ctx := newTestContext(map[Var]any{"foo": int64(422)})
		ctx.currResponse = &http.Response{StatusCode: http.StatusOK}
		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet", func(t *testing.T) {
		exp := ExpectStatusCodeIn(http.StatusOK)
		ctx := newTestContext(nil)
		ctx.currResponse = &http.Response{StatusCode: http.StatusNotFound}
		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		umerr := unmet.(UnmetError)
		assert.NoError(t, umerr.Expected().CoercionError)
		assert.NoError(t, err)
	})
	t.Run("unmet - bad type", func(t *testing.T) {
		exp := ExpectStatusCodeIn(true)
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		umerr := unmet.(UnmetError)
		assert.Error(t, umerr.Expected().CoercionError)
		assert.NoError(t, err)
	})
	t.Run("resolve error", func(t *testing.T) {
		exp := ExpectStatusCodeIn(Var("foo"))
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.Error(t, err)
	})
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

func Test_contains(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		exp := &contains{
			value:    "foo",
			contains: "bar",
			frame:    framing.NewFrame(0),
		}
		assert.Equal(t, "Expect contains: \"bar\"", exp.Name())
		assert.NotNil(t, exp.Frame())
	})
	t.Run("met", func(t *testing.T) {
		exp := &contains{
			value:    "foo",
			contains: "fo",
			frame:    framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("met (map)", func(t *testing.T) {
		exp := &contains{
			value:    map[string]any{"foo": nil},
			contains: `{"foo"`,
			frame:    framing.NewFrame(0),
		}
		unmet, err := exp.Met(nil)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("missing var", func(t *testing.T) {
		exp := &contains{
			value:    Var("foo"),
			contains: "bar",
			frame:    framing.NewFrame(0),
		}
		_, err := exp.Met(newTestContext(nil))
		assert.Error(t, err)
	})
	t.Run("unmet (int var)", func(t *testing.T) {
		exp := &contains{
			value:    Var("foo"),
			contains: "99",
			frame:    framing.NewFrame(0),
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
		exp := &contains{
			value:    map[string]any{"x": &unmarshalable{}},
			contains: "bar",
			frame:    framing.NewFrame(0),
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

func Test_expectMockCall(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		exp := &expectMockCall{
			name:   "mock",
			path:   "/foos",
			method: http.MethodGet,
			frame:  framing.NewFrame(0),
		}
		assert.Equal(t, "EXPECT MOCK SERVICE CALL [mock]: GET /foos", exp.Name())
		assert.NotNil(t, exp.Frame())
	})
	t.Run("met", func(t *testing.T) {
		exp := &expectMockCall{
			name:   "mock",
			path:   "/foos",
			method: http.MethodGet,
			frame:  framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		ms := &mockMockedService{called: true}
		ctx.mockServices["mock"] = ms

		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet", func(t *testing.T) {
		exp := &expectMockCall{
			name:   "mock",
			path:   "/foos",
			method: http.MethodGet,
			frame:  framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		ms := &mockMockedService{called: false}
		ctx.mockServices["mock"] = ms

		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("missing var in path", func(t *testing.T) {
		exp := &expectMockCall{
			name:   "mock",
			path:   "/foos/{$id}",
			method: http.MethodGet,
			frame:  framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		ms := &mockMockedService{called: false}
		ctx.mockServices["mock"] = ms

		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unresolved variables in string ")
	})
	t.Run("unknown mock service", func(t *testing.T) {
		exp := &expectMockCall{
			name:   "mock",
			path:   "/foos",
			method: http.MethodGet,
			frame:  framing.NewFrame(0),
		}
		ctx := newTestContext(nil)

		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown mock service ")
	})
}

func Test_propertiesCheck(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		exp := &propertiesCheck{
			frame: framing.NewFrame(0),
		}
		assert.Equal(t, "Expect Properties", exp.Name())
		assert.NotNil(t, exp.Frame())

		exp.only = true
		assert.Equal(t, "Expect Only Properties", exp.Name())
	})
	t.Run("met", func(t *testing.T) {
		exp := &propertiesCheck{
			value:      map[string]any{"foo": nil, "bar": nil},
			properties: []string{"foo", "bar"},
			frame:      framing.NewFrame(0),
		}
		ctx := newTestContext(nil)

		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("met - general map", func(t *testing.T) {
		exp := &propertiesCheck{
			value:      map[string]bool{"foo": true, "bar": true},
			properties: []string{"foo", "bar"},
			frame:      framing.NewFrame(0),
		}
		ctx := newTestContext(nil)

		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet - missing property", func(t *testing.T) {
		exp := &propertiesCheck{
			value:      map[string]any{"foo": nil},
			properties: []string{"foo", "bar"},
			frame:      framing.NewFrame(0),
		}
		ctx := newTestContext(nil)

		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet - only, extra property", func(t *testing.T) {
		exp := &propertiesCheck{
			value:      map[string]any{"foo": nil, "bar": nil, "baz": nil},
			properties: []string{"foo", "bar"},
			only:       true,
			frame:      framing.NewFrame(0),
		}
		ctx := newTestContext(nil)

		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet - invalid type", func(t *testing.T) {
		exp := &propertiesCheck{
			value:      "not a map",
			properties: []string{"foo", "bar"},
			frame:      framing.NewFrame(0),
		}
		ctx := newTestContext(nil)

		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.Contains(t, unmet.Error(), "cannot check properties on ")
		assert.NoError(t, err)
	})
}

func TestFail(t *testing.T) {
	exp := &failCheck{
		msg:   "fooey",
		frame: framing.NewFrame(0),
	}
	ctx := newTestContext(nil)
	unmet, err := exp.Met(ctx)
	assert.NoError(t, unmet)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fooey")
}

func TestExpectVarSet(t *testing.T) {
	t.Run("met", func(t *testing.T) {
		exp := &varCheck{
			name:  "foo",
			frame: framing.NewFrame(0),
		}
		ctx := newTestContext(map[Var]any{"foo": nil})
		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet", func(t *testing.T) {
		exp := &varCheck{
			name:  "foo",
			frame: framing.NewFrame(0),
		}
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
}

func TestExpectTrueFalse(t *testing.T) {
	t.Run("true met bool", func(t *testing.T) {
		exp := ExpectTrue(true)
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("false met bool", func(t *testing.T) {
		exp := ExpectFalse(false)
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("true unmet bool", func(t *testing.T) {
		exp := ExpectTrue(false)
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("false unmet bool", func(t *testing.T) {
		exp := ExpectFalse(true)
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("unmet not a bool", func(t *testing.T) {
		exp := ExpectTrue("not a bool")
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("true expectation met", func(t *testing.T) {
		exp := ExpectTrue(ExpectEqual(0, 0))
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("true expectation unmet", func(t *testing.T) {
		exp := ExpectTrue(ExpectEqual(1, 0))
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("false expectation unmet", func(t *testing.T) {
		exp := ExpectFalse(ExpectEqual(1, 0))
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.NoError(t, unmet)
		assert.NoError(t, err)
	})
	t.Run("false expectation met", func(t *testing.T) {
		exp := ExpectFalse(ExpectEqual(0, 0))
		ctx := newTestContext(nil)
		unmet, err := exp.Met(ctx)
		assert.Error(t, unmet)
		assert.NoError(t, err)
	})
}

func TestExpectationFuncs(t *testing.T) {
	testCases := []struct {
		value      Expectation
		expectName string
	}{
		{
			value:      ExpectOK(),
			expectName: "Expect OK",
		},
		{
			value:      ExpectCreated(),
			expectName: "Expect Created",
		},
		{
			value:      ExpectAccepted(),
			expectName: "Expect Accepted",
		},
		{
			value:      ExpectNoContent(),
			expectName: "Expect No Content",
		},
		{
			value:      ExpectBadRequest(),
			expectName: "Expect Bad Request",
		},
		{
			value:      ExpectUnauthorized(),
			expectName: "Expect Unauthorized",
		},
		{
			value:      ExpectForbidden(),
			expectName: "Expect Forbidden",
		},
		{
			value:      ExpectNotFound(),
			expectName: "Expect Not Found",
		},
		{
			value:      ExpectConflict(),
			expectName: "Expect Conflict",
		},
		{
			value:      ExpectGone(),
			expectName: "Expect Gone",
		},
		{
			value:      ExpectUnprocessableEntity(),
			expectName: "Expect Unprocessable Entity",
		},
		{
			value:      ExpectStatus(0),
			expectName: "Expect Status Code",
		},
		{
			value:      ExpectMatch(nil, "foo"),
			expectName: "Expect match: \"foo\"",
		},
		{
			value:      ExpectContains(nil, "foo"),
			expectName: "Expect contains: \"foo\"",
		},
		{
			value:      ExpectType(nil, Type[string]()),
			expectName: "Expect type: string",
		},
		{
			value:      ExpectNil(nil),
			expectName: "Expect Nil",
		},
		{
			value:      ExpectNotNil(nil),
			expectName: "Expect Not Nil",
		},
		{
			value:      ExpectLen(nil, 0),
			expectName: "Expect Len",
		},
		{
			value:      ExpectMockServiceCalled("svc", "/api", GET),
			expectName: "EXPECT MOCK SERVICE CALL [svc]: GET /api",
		},
		{
			value:      ExpectHasProperties(nil),
			expectName: "Expect Properties",
		},
		{
			value:      ExpectOnlyHasProperties(nil),
			expectName: "Expect Only Properties",
		},
		{
			value:      Fail("fooey"),
			expectName: "FAIL \"fooey\"",
		},
		{
			value:      ExpectVarSet(Var("foo")),
			expectName: "Expect Var(\"foo\") set",
		},
		{
			value:      ExpectTrue(Var("foo")),
			expectName: "Expect True(Var(foo))",
		},
		{
			value:      ExpectTrue(ExpectEqual(0, 1)),
			expectName: "Expect True(ExpectEqual)",
		},
		{
			value:      ExpectFalse(Var("foo")),
			expectName: "Expect False(Var(foo))",
		},
		{
			value:      ExpectFalse(ExpectEqual(0, 1)),
			expectName: "Expect False(ExpectEqual)",
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			assert.Equal(t, tc.expectName, tc.value.Name())
			assert.NotNil(t, tc.value.Frame())
			assert.False(t, tc.value.IsRequired())
		})
	}
}
