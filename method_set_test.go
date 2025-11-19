package marrow

import (
	"github.com/go-andiamo/marrow/coverage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
)

func TestSetQueryParam(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c := SetQueryParam("foo", "bar")
		assert.Equal(t, `SetQueryParam("foo")`, c.Name())
		assert.NotNil(t, c.Frame())
	})
	t.Run("run ok", func(t *testing.T) {
		m := Method(GET, "").If(Before, Var("yes"), SetQueryParam("foo", Var("foo")))
		ctx := newTestContext(map[Var]any{"yes": true, "foo": "bar"})
		ctx.httpDo = &dummyDo{status: http.StatusOK, body: []byte(`{"foo":"bar"}`)}
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.setCurrentEndpoint(Endpoint("/api", ""))
		err := m.Run(ctx)
		require.NoError(t, err)
		require.False(t, cov.HasFailures())
		assert.Len(t, cov.Failures, 0)
		assert.Len(t, cov.Unmet, 0)
		assert.Len(t, cov.Timings, 1)
		assert.Equal(t, "/api?foo=bar", cov.Timings[0].Request.URL.String())
	})
	t.Run("run too late (after)", func(t *testing.T) {
		m := Method(GET, "").If(After, Var("yes"), SetQueryParam("foo", "bar"))
		ctx := newTestContext(map[Var]any{"yes": true})
		ctx.httpDo = &dummyDo{status: http.StatusOK, body: []byte(`{"foo":"bar"}`)}
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.setCurrentEndpoint(Endpoint("/api", ""))
		err := m.Run(ctx)
		require.NoError(t, err)
		require.True(t, cov.HasFailures())
		assert.Len(t, cov.Failures, 1)
	})
	t.Run("method not set", func(t *testing.T) {
		c := SetQueryParam("foo", "bar")
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "method not set")
	})
}

func TestSetRequestHeader(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c := SetRequestHeader("foo", "bar")
		assert.Equal(t, `SetRequestHeader("foo")`, c.Name())
		assert.NotNil(t, c.Frame())
	})
	t.Run("run ok", func(t *testing.T) {
		m := Method(GET, "").If(Before, Var("yes"), SetRequestHeader("foo", Var("foo")))
		ctx := newTestContext(map[Var]any{"yes": true, "foo": "bar"})
		ctx.httpDo = &dummyDo{status: http.StatusOK, body: []byte(`{"foo":"bar"}`)}
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.setCurrentEndpoint(Endpoint("/api", ""))
		err := m.Run(ctx)
		require.NoError(t, err)
		require.False(t, cov.HasFailures())
		assert.Len(t, cov.Failures, 0)
		assert.Len(t, cov.Unmet, 0)
		assert.Len(t, cov.Timings, 1)
		assert.Equal(t, "bar", cov.Timings[0].Request.Header.Get("foo"))
	})
	t.Run("run too late (after)", func(t *testing.T) {
		m := Method(GET, "").If(After, Var("yes"), SetRequestHeader("foo", "bar"))
		ctx := newTestContext(map[Var]any{"yes": true})
		ctx.httpDo = &dummyDo{status: http.StatusOK, body: []byte(`{"foo":"bar"}`)}
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.setCurrentEndpoint(Endpoint("/api", ""))
		err := m.Run(ctx)
		require.NoError(t, err)
		require.True(t, cov.HasFailures())
		assert.Len(t, cov.Failures, 1)
	})
	t.Run("method not set", func(t *testing.T) {
		c := SetRequestHeader("foo", "bar")
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "method not set")
	})
}

func TestSetRequestBody(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c := SetRequestBody("foo")
		assert.Equal(t, `SetRequestBody()`, c.Name())
		assert.NotNil(t, c.Frame())
	})
	t.Run("run ok", func(t *testing.T) {
		m := Method(GET, "").If(Before, Var("yes"), SetRequestBody(Var("foo")), SetRequestHeader("foo", "bar"))
		ctx := newTestContext(map[Var]any{"yes": true, "foo": "bar"})
		ctx.httpDo = &dummyDo{status: http.StatusOK, body: []byte(`{"foo":"bar"}`)}
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.setCurrentEndpoint(Endpoint("/api", ""))
		err := m.Run(ctx)
		require.NoError(t, err)
		rawM := m.(*method)
		assert.Equal(t, Var("foo"), rawM.body)
		require.False(t, cov.HasFailures())
		assert.Len(t, cov.Failures, 0)
		assert.Len(t, cov.Unmet, 0)
		assert.Len(t, cov.Timings, 1)
		assert.Equal(t, "bar", cov.Timings[0].Request.Header.Get("foo"))
	})
	t.Run("run too late (after)", func(t *testing.T) {
		m := Method(GET, "").If(After, Var("yes"), SetRequestBody("foo"))
		ctx := newTestContext(map[Var]any{"yes": true})
		ctx.httpDo = &dummyDo{status: http.StatusOK, body: []byte(`{"foo":"bar"}`)}
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.setCurrentEndpoint(Endpoint("/api", ""))
		err := m.Run(ctx)
		require.NoError(t, err)
		require.True(t, cov.HasFailures())
		assert.Len(t, cov.Failures, 1)
	})
	t.Run("method not set", func(t *testing.T) {
		c := SetRequestBody("foo")
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "method not set")
	})
}

func TestSetRequestUseCookie(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c := SetRequestUseCookie("foo")
		assert.Equal(t, `SetRequestUseCookie("foo")`, c.Name())
		assert.NotNil(t, c.Frame())
	})
	t.Run("run ok", func(t *testing.T) {
		m := Method(GET, "").If(Before, Var("yes"), SetRequestUseCookie("session"), SetRequestHeader("foo", "bar"))
		ctx := newTestContext(map[Var]any{"yes": true, "foo": "bar"})
		ctx.httpDo = &dummyDo{status: http.StatusOK, body: []byte(`{"foo":"bar"}`)}
		ctx.cookieJar["session"] = &http.Cookie{Name: "session", Value: "foo"}
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.setCurrentEndpoint(Endpoint("/api", ""))
		err := m.Run(ctx)
		require.NoError(t, err)
		rawM := m.(*method)
		assert.Len(t, rawM.useCookies, 1)
		require.False(t, cov.HasFailures())
		assert.Len(t, cov.Failures, 0)
		assert.Len(t, cov.Unmet, 0)
		assert.Len(t, cov.Timings, 1)
		assert.Len(t, cov.Timings[0].Request.Cookies(), 1)
		assert.Equal(t, "bar", cov.Timings[0].Request.Header.Get("foo"))
	})
	t.Run("run too late (after)", func(t *testing.T) {
		m := Method(GET, "").If(After, Var("yes"), SetRequestUseCookie("foo"))
		ctx := newTestContext(map[Var]any{"yes": true})
		ctx.httpDo = &dummyDo{status: http.StatusOK, body: []byte(`{"foo":"bar"}`)}
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		ctx.setCurrentEndpoint(Endpoint("/api", ""))
		err := m.Run(ctx)
		require.NoError(t, err)
		require.True(t, cov.HasFailures())
		assert.Len(t, cov.Failures, 1)
	})
	t.Run("method not set", func(t *testing.T) {
		c := SetRequestUseCookie("foo")
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "method not set")
	})
}
