package marrow

import (
	"github.com/go-andiamo/marrow/coverage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewEndpoint(t *testing.T) {
	e := Endpoint("/api/foos/{id}", "Foos")
	require.NotNil(t, e)
	f := e.Frame()
	assert.NotNil(t, f)
	assert.Equal(t, t.Name(), f.Name)

	t.Run("basic", func(t *testing.T) {
		e := Endpoint("/api/foos/{id}", "Foos")
		require.NotNil(t, e)
		assert.Equal(t, "/api/foos/{id}", e.Url())
		assert.Equal(t, "Foos", e.Description())
		assert.Equal(t, "/api/foos/{id} \"Foos\"", e.String())

		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.methods, 0)
		assert.Len(t, raw.befores, 0)
		assert.Len(t, raw.afters, 0)
	})
	t.Run("with method", func(t *testing.T) {
		e := Endpoint("/api/foos/{id}", "Foos",
			nil, Method(GET, ""))
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.methods, 1)
	})
	t.Run("with methods", func(t *testing.T) {
		ms := []Method_{
			Method(GET, ""),
			Method(POST, ""),
			nil,
		}
		e := Endpoint("/api/foos/{id}", "Foos", ms)
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.methods, 2)
	})
	t.Run("with before/after", func(t *testing.T) {
		e := Endpoint("/api/foos/{id}", "Foos",
			DoBefore(SetVar("foo", nil)), DoAfter(SetVar("foo", nil)), nil)
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.befores, 1)
		assert.Len(t, raw.afters, 1)
	})
	t.Run("with before/after slice", func(t *testing.T) {
		e := Endpoint("/api/foos/{id}", "Foos",
			[]BeforeAfter{DoBefore(SetVar("foo", nil)), DoAfter(SetVar("foo", nil)), nil})
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.befores, 1)
		assert.Len(t, raw.afters, 1)
	})
	t.Run("with endpoint", func(t *testing.T) {
		e := Endpoint("/api/foos", "Foos", Endpoint("/{id}", ""))
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.subs, 1)
	})
	t.Run("with endpoints", func(t *testing.T) {
		subs := []Endpoint_{
			Endpoint("/{id}", ""),
			nil,
			Endpoint("/{id}", ""),
		}
		e := Endpoint("/api/foos", "Foos", subs)
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.subs, 2)
	})
	t.Run("with mixed", func(t *testing.T) {
		ops := []any{
			DoBefore(SetVar("foo", nil)),
			DoAfter(SetVar("foo", nil)),
			nil,
			Method(GET, ""),
			Endpoint("/{id}", ""),
		}
		e := Endpoint("/api/foos", "Foos", ops)
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.methods, 1)
		assert.Len(t, raw.befores, 1)
		assert.Len(t, raw.afters, 1)
		assert.Len(t, raw.subs, 1)
	})
	t.Run("panics with unsupported op type", func(t *testing.T) {
		require.Panics(t, func() {
			_ = Endpoint("/api/foos/{id}", "Foos", "not a valid option")
		})
		require.Panics(t, func() {
			_ = Endpoint("/api/foos/{id}", "Foos", []any{"not a valid option"})
		})
	})
}

func TestEndpoint_Url_WithAncestors(t *testing.T) {
	t.Run("with ancestors list", func(t *testing.T) {
		e := Endpoint("/{id}", "")
		assert.Equal(t, "/{id}", e.Url())
		e.setAncestry([]Endpoint_{
			Endpoint("/api", ""),
			Endpoint("/foos", ""),
		})
		assert.Equal(t, "/api/foos/{id}", e.Url())
	})
}

func TestEndpoint_Run(t *testing.T) {
	e := Endpoint("/api", "",
		DoBefore(SetVar("foo", nil)),
		DoAfter(SetVar("foo", nil)),
		Method(GET, ""),
		Endpoint("/foos", "",
			Method(GET, ""),
			Endpoint("/bars", "",
				Method(GET, ""),
			),
		),
	)
	ctx := newTestContext(nil)
	cov := coverage.NewCoverage()
	ctx.coverage = cov
	ctx.httpDo = &dummyDo{status: 200, body: []byte(`{}`)}
	err := e.Run(ctx)
	require.NoError(t, err)
	assert.Len(t, cov.Failures, 0)
}

func TestEndpoint_Run_WithFailures(t *testing.T) {
	t.Run("before fails", func(t *testing.T) {
		e := Endpoint("/api", "",
			DoBefore(SetVar("foo", Var("bar"))),
			Method(GET, "").SetVar(Before, "foo", Var("bar")),
			Endpoint("/foos", "", DoBefore(SetVar("foo", Var("bar")))),
			DoAfter(SetVar("foo", Var("bar"))),
		)
		ctx := newTestContext(nil)
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.httpDo = &dummyDo{status: 200, body: []byte(`{}`)}
		err := e.Run(ctx)
		require.NoError(t, err)
		assert.Len(t, cov.Failures, 1)
		assert.True(t, ctx.failed)
	})
	t.Run("method fails", func(t *testing.T) {
		e := Endpoint("/api", "",
			Method(GET, "").SetVar(Before, "foo", Var("bar")),
			Endpoint("/foos", "", DoBefore(SetVar("foo", Var("bar")))),
			DoAfter(SetVar("foo", Var("bar"))),
		)
		ctx := newTestContext(nil)
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.httpDo = &dummyDo{status: 200}
		err := e.Run(ctx)
		require.NoError(t, err)
		assert.Len(t, cov.Failures, 1)
		assert.True(t, ctx.failed)
	})
	t.Run("sub-endpoint fails", func(t *testing.T) {
		e := Endpoint("/api", "",
			Endpoint("/foos", "", DoBefore(SetVar("foo", Var("bar")))),
			DoAfter(SetVar("foo", Var("bar"))),
		)
		ctx := newTestContext(nil)
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.httpDo = &dummyDo{status: 200}
		err := e.Run(ctx)
		require.NoError(t, err)
		assert.Len(t, cov.Failures, 1)
		assert.True(t, ctx.failed)
	})
	t.Run("after fails", func(t *testing.T) {
		e := Endpoint("/api", "",
			DoAfter(SetVar("foo", Var("bar"))),
		)
		ctx := newTestContext(nil)
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.httpDo = &dummyDo{status: 200, body: []byte(`{}`)}
		err := e.Run(ctx)
		require.NoError(t, err)
		assert.Len(t, cov.Failures, 1)
		assert.True(t, ctx.failed)
	})
}
