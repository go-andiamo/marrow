package marrow

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestMethod(t *testing.T) {
	m := Method(GET, "foo")
	assert.NotNil(t, m.Frame())
	assert.Equal(t, "GET \"foo\"", m.String())
	assert.Equal(t, "GET", m.MethodName())
	assert.Equal(t, "foo", m.Description())

	m.AssertOK()
	raw := m.(*method)
	assert.Len(t, raw.expectations, 1)
	exp := raw.expectations[0]
	assert.Equal(t, "Expect OK", exp.Name())
	assert.False(t, raw.failFast)
	m.FailFast()
	assert.True(t, raw.failFast)
}

func TestMethod_WithBeforesAndAfters(t *testing.T) {
	m := Method(GET, "foo",
		SetVar(Before, "foo", "bar"),
		ClearVars(After),
	).AssertOK().SetVar(After, "bar", 42).ClearVars(Before).AssertEqual("foo", "bar")
	raw := m.(*method)
	assert.Len(t, raw.preCaptures, 2)
	assert.Len(t, raw.expectations, 2)
	assert.Len(t, raw.postCaptures, 2)
	assert.Len(t, raw.postOps, 4)
	assert.False(t, raw.postOps[0].isExpectation)
	assert.Equal(t, 0, raw.postOps[0].index)
	assert.True(t, raw.postOps[1].isExpectation)
	assert.Equal(t, 0, raw.postOps[1].index)
	assert.False(t, raw.postOps[2].isExpectation)
	assert.Equal(t, 1, raw.postOps[2].index)
	assert.True(t, raw.postOps[3].isExpectation)
	assert.Equal(t, 1, raw.postOps[3].index)
}

func TestMethod_BuildRequest(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := Method("", "")
		m.PathParam(Var("foo")).PathParam(Var("bar"))
		m.QueryParam("q1", nil).QueryParam("q2", true).QueryParam("q3", Var("q3"))
		m.RequestHeader("X-Foo", Var("foo"))
		m.RequestBody(JSON{
			"foo": Var("bar"),
		})
		m.UseCookie("session")
		ctx := newContext(map[Var]any{
			"foo":  "aaa",
			"bar":  Var("bar2"),
			"bar2": 42,
			"q3":   "foo??",
		})
		ctx.StoreCookie(&http.Cookie{Name: "session", Value: "test"})
		ctx.host = "http://localhost:8080"
		ctx.currEndpoint = Endpoint("/foos/{id}/bars/{id}", "").(*endpoint)

		raw := m.(*method)
		req, ok := raw.buildRequest(ctx)
		require.True(t, ok)
		assert.NotNil(t, req)
		assert.Equal(t, "http://localhost:8080/foos/aaa/bars/42?q1&q2=true&q3=foo%3F%3F", req.URL.String())
		assert.Equal(t, "aaa", req.Header.Get("X-Foo"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		data, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		assert.Equal(t, "{\"foo\":42}", string(data))
		c, err := req.Cookie("session")
		require.NoError(t, err)
		assert.Equal(t, "test", c.Value)
	})
	t.Run("header fail", func(t *testing.T) {
		m := Method("", "")
		m.RequestHeader("X-Foo", Var("foo"))
		ctx := newContext(nil)
		ctx.currEndpoint = Endpoint("/foos", "").(*endpoint)
		raw := m.(*method)
		_, ok := raw.buildRequest(ctx)
		require.False(t, ok)
	})
}

func TestMethod_BuildRequestUrl(t *testing.T) {
	m := Method(GET, "")
	m.PathParam(Var("foo")).PathParam(Var("bar"))
	m.QueryParam("q1", nil).QueryParam("q2", true).QueryParam("q3", Var("q3"))
	ctx := newContext(map[Var]any{
		"foo":  "aaa",
		"bar":  Var("bar2"),
		"bar2": 42,
		"q3":   "foo??",
	})
	ctx.host = "http://localhost:8080"
	ctx.currEndpoint = Endpoint("/foos/{id}/bars/{id}", "").(*endpoint)

	raw := m.(*method)
	url, err := raw.buildRequestUrl(ctx)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/foos/aaa/bars/42?q1&q2=true&q3=foo%3F%3F", url)
}

func TestMethod_BuildRequestBody(t *testing.T) {
	t.Run("default marshal", func(t *testing.T) {
		m := Method(GET, "")
		m.RequestBody(map[string]any{
			"foo": Var("foo"),
		})
		ctx := newContext(map[Var]any{
			"foo": 42,
		})
		raw := m.(*method)
		body, err := raw.buildRequestBody(ctx)
		require.NoError(t, err)
		data, err := io.ReadAll(body)
		require.NoError(t, err)
		assert.Equal(t, "{\"foo\":42}", string(data))
	})
	t.Run("custom marshal", func(t *testing.T) {
		m := Method(GET, "")
		m.RequestBody("foo")
		m.RequestMarshal(func(ctx Context, body any) ([]byte, error) {
			return []byte("custom"), nil
		})
		ctx := newContext(nil)
		raw := m.(*method)
		body, err := raw.buildRequestBody(ctx)
		require.NoError(t, err)
		data, err := io.ReadAll(body)
		require.NoError(t, err)
		assert.Equal(t, "custom", string(data))
	})
}

func Test_normalizeBody(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var body any
		normalizedBody, err := normalizeBody(body)
		require.NoError(t, err)
		assert.Nil(t, normalizedBody)
	})
	t.Run("string", func(t *testing.T) {
		var body any
		data := []byte(`"some string"`)
		d := json.NewDecoder(bytes.NewReader(data))
		d.UseNumber()
		err := d.Decode(&body)
		require.NoError(t, err)
		normalizedBody, err := normalizeBody(body)
		require.NoError(t, err)
		assert.NotNil(t, normalizedBody)
		assert.Equal(t, "some string", normalizedBody)
	})
	t.Run("number", func(t *testing.T) {
		var body any
		data := []byte(`42`)
		d := json.NewDecoder(bytes.NewReader(data))
		d.UseNumber()
		err := d.Decode(&body)
		require.NoError(t, err)
		normalizedBody, err := normalizeBody(body)
		require.NoError(t, err)
		assert.NotNil(t, normalizedBody)
		assert.Equal(t, int64(42), normalizedBody)
	})
	t.Run("map", func(t *testing.T) {
		var body any
		data := []byte(`{"foo":42,"bar":2.2}`)
		d := json.NewDecoder(bytes.NewReader(data))
		d.UseNumber()
		err := d.Decode(&body)
		require.NoError(t, err)
		normalizedBody, err := normalizeBody(body)
		require.NoError(t, err)
		assert.NotNil(t, normalizedBody)
		assert.Equal(t, int64(42), normalizedBody.(map[string]any)["foo"])
		assert.Equal(t, 2.2, normalizedBody.(map[string]any)["bar"])
	})
	t.Run("map mixed", func(t *testing.T) {
		body := map[string]any{
			"foo": json.Number("42"),
			"bar": map[string]any{"foo": json.Number("42")},
			"baz": []any{"buzz", json.Number("42")},
		}
		normalizedBody, err := normalizeBody(body)
		require.NoError(t, err)
		m, ok := normalizedBody.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, int64(42), m["foo"])
		assert.Equal(t, map[string]any{"foo": int64(42)}, m["bar"])
		assert.Equal(t, []any{"buzz", int64(42)}, m["baz"])
	})
	t.Run("map,map errors", func(t *testing.T) {
		body := map[string]any{
			"foo": map[string]any{"foo": json.Number("invalid number")},
		}
		_, err := normalizeBody(body)
		require.Error(t, err)
	})
	t.Run("map,slice errors", func(t *testing.T) {
		body := map[string]any{
			"foo": []any{json.Number("invalid number")},
		}
		_, err := normalizeBody(body)
		require.Error(t, err)
	})
	t.Run("slice", func(t *testing.T) {
		var body any
		data := []byte(`[42,2.2]`)
		d := json.NewDecoder(bytes.NewReader(data))
		d.UseNumber()
		err := d.Decode(&body)
		require.NoError(t, err)
		normalizedBody, err := normalizeBody(body)
		require.NoError(t, err)
		assert.NotNil(t, normalizedBody)
		assert.Equal(t, int64(42), normalizedBody.([]any)[0])
		assert.Equal(t, 2.2, normalizedBody.([]any)[1])
	})
	t.Run("slice mixed", func(t *testing.T) {
		body := []any{
			json.Number("42"),
			map[string]any{"foo": json.Number("42")},
			[]any{"buzz", json.Number("42")},
		}
		normalizedBody, err := normalizeBody(body)
		require.NoError(t, err)
		sl, ok := normalizedBody.([]any)
		require.True(t, ok)
		assert.Equal(t, int64(42), sl[0])
		assert.Equal(t, map[string]any{"foo": int64(42)}, sl[1])
		assert.Equal(t, []any{"buzz", int64(42)}, sl[2])
	})
	t.Run("slice,map errors", func(t *testing.T) {
		body := []any{
			map[string]any{"foo": json.Number("invalid number")},
		}
		_, err := normalizeBody(body)
		require.Error(t, err)
	})
	t.Run("slice,slice errors", func(t *testing.T) {
		body := []any{
			[]any{json.Number("invalid number")},
		}
		_, err := normalizeBody(body)
		require.Error(t, err)
	})
	t.Run("number error", func(t *testing.T) {
		body := json.Number("invalid number")
		_, err := normalizeBody(body)
		require.Error(t, err)
	})
	t.Run("map error", func(t *testing.T) {
		body := map[string]any{"foo": json.Number("invalid number")}
		_, err := normalizeBody(body)
		require.Error(t, err)
	})
	t.Run("slice error", func(t *testing.T) {
		body := []any{json.Number("invalid number")}
		_, err := normalizeBody(body)
		require.Error(t, err)
	})
}

func TestMethod_unmarshalResponseBody(t *testing.T) {
	t.Run("nil response body", func(t *testing.T) {
		response := &http.Response{}
		ctx := newContext(nil)
		m := &method{}
		ok := m.unmarshalResponseBody(ctx, response)
		require.True(t, ok)
		assert.Nil(t, ctx.currBody)
	})
	t.Run("default unmarshal", func(t *testing.T) {
		response := &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(`{"foo":42}`))),
		}
		ctx := newContext(nil)
		m := &method{}
		ok := m.unmarshalResponseBody(ctx, response)
		require.True(t, ok)
		assert.NotNil(t, ctx.currBody)
		assert.Equal(t, map[string]any{"foo": int64(42)}, ctx.currBody)
	})
	t.Run("default unmarshal errors", func(t *testing.T) {
		response := &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(`{invalid json`))),
		}
		ctx := newContext(nil)
		m := &method{}
		ok := m.unmarshalResponseBody(ctx, response)
		require.False(t, ok)
		require.True(t, ctx.failed)
	})
	t.Run("custom unmarshal", func(t *testing.T) {
		response := &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(`{"foo":42}`))),
		}
		ctx := newContext(nil)
		m := &method{}
		m.ResponseUnmarshal(func(response *http.Response) (any, error) {
			var body any
			err := json.NewDecoder(response.Body).Decode(&body)
			return body, err
		})
		ok := m.unmarshalResponseBody(ctx, response)
		require.True(t, ok)
		assert.NotNil(t, ctx.currBody)
		assert.Equal(t, map[string]any{"foo": float64(42)}, ctx.currBody)
	})
	t.Run("custom unmarshal errors", func(t *testing.T) {
		response := &http.Response{
			Body: io.NopCloser(bytes.NewReader([]byte(`{"foo":42}`))),
		}
		ctx := newContext(nil)
		m := &method{}
		m.ResponseUnmarshal(func(response *http.Response) (any, error) {
			return nil, errors.New("fooey")
		})
		ok := m.unmarshalResponseBody(ctx, response)
		require.False(t, ok)
		require.True(t, ctx.failed)
	})
}

func TestMethod_Run(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ctx := newContext(nil)
		ctx.httpDo = &dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo":42}`),
		}
		ctx.currEndpoint = Endpoint("/foos/{id}", "")
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		m := Method(GET, "",
			SetVar(Before, "id", 123),
		).PathParam(Var("id")).
			AssertOK().
			SetVar(After, "foo", JsonPath(Body, "foo")).
			AssertEqual(Var("foo"), 42)

		err := m.Run(ctx)
		require.NoError(t, err)
		require.False(t, ctx.failed)
		assert.Len(t, cov.Timings, 1)
		assert.Len(t, cov.Met, 2)
		require.Len(t, cov.Endpoints, 1)
		ec, ok := cov.Endpoints["/foos/{id}"]
		require.True(t, ok)
		assert.Len(t, ec.Timings, 1)
		assert.Len(t, ec.Met, 2)
		require.Len(t, ec.Methods, 1)
		mc, ok := ec.Methods["GET"]
		require.True(t, ok)
		assert.Len(t, mc.Timings, 1)
		assert.Len(t, mc.Met, 2)
	})
	t.Run("pre-capture fails", func(t *testing.T) {
		ctx := newContext(nil)
		ctx.currEndpoint = Endpoint("/foos", "")
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		m := Method(GET, "", SetVar(Before, "id", Var("missing")))
		err := m.Run(ctx)
		require.NoError(t, err)
		require.True(t, ctx.failed)
		assert.Len(t, cov.Failures, 1)
		require.Len(t, cov.Endpoints, 1)
		ec, ok := cov.Endpoints["/foos"]
		require.True(t, ok)
		assert.Len(t, ec.Failures, 1)
		require.Len(t, ec.Methods, 1)
		mc, ok := ec.Methods["GET"]
		require.True(t, ok)
		assert.Len(t, mc.Failures, 1)
	})
	t.Run("post-capture fails", func(t *testing.T) {
		ctx := newContext(nil)
		ctx.httpDo = &dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo":42}`),
		}
		ctx.currEndpoint = Endpoint("/foos", "")
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		m := Method(GET, "").FailFast().AssertOK().
			SetVar(After, "foo", Var("missing")).
			AssertNotFound()
		err := m.Run(ctx)
		require.NoError(t, err)
		require.True(t, ctx.failed)
		assert.Len(t, cov.Timings, 1)
		assert.Len(t, cov.Failures, 1)
		assert.Len(t, cov.Met, 1)
		assert.Len(t, cov.Skipped, 1)
		require.Len(t, cov.Endpoints, 1)
		ec, ok := cov.Endpoints["/foos"]
		require.True(t, ok)
		assert.Len(t, ec.Timings, 1)
		assert.Len(t, ec.Failures, 1)
		assert.Len(t, ec.Met, 1)
		assert.Len(t, ec.Skipped, 1)
		require.Len(t, ec.Methods, 1)
		mc, ok := ec.Methods["GET"]
		require.True(t, ok)
		assert.Len(t, ec.Timings, 1)
		assert.Len(t, mc.Failures, 1)
		assert.Len(t, ec.Met, 1)
		assert.Len(t, ec.Skipped, 1)
	})
	t.Run("expectation met & unmet", func(t *testing.T) {
		ctx := newContext(nil)
		ctx.httpDo = &dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo":42}`),
		}
		ctx.currEndpoint = Endpoint("/foos", "")
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		m := Method(GET, "").AssertOK().AssertNotFound()
		err := m.Run(ctx)
		require.NoError(t, err)
		require.False(t, ctx.failed)
		assert.Len(t, cov.Timings, 1)
		assert.Len(t, cov.Unmet, 1)
		assert.Len(t, cov.Met, 1)
		require.Len(t, cov.Endpoints, 1)
		ec, ok := cov.Endpoints["/foos"]
		require.True(t, ok)
		assert.Len(t, ec.Timings, 1)
		assert.Len(t, ec.Unmet, 1)
		assert.Len(t, ec.Met, 1)
		require.Len(t, ec.Methods, 1)
		mc, ok := ec.Methods["GET"]
		require.True(t, ok)
		assert.Len(t, ec.Timings, 1)
		assert.Len(t, mc.Unmet, 1)
		assert.Len(t, ec.Met, 1)
	})
	t.Run("expectation unmet - fail fast", func(t *testing.T) {
		ctx := newContext(nil)
		ctx.httpDo = &dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo":42}`),
		}
		ctx.currEndpoint = Endpoint("/foos", "")
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		m := Method(GET, "").FailFast().AssertNotFound().AssertOK()
		err := m.Run(ctx)
		require.NoError(t, err)
		require.False(t, ctx.failed)
		assert.Len(t, cov.Timings, 1)
		assert.Len(t, cov.Unmet, 1)
		assert.Len(t, cov.Skipped, 1)
		require.Len(t, cov.Endpoints, 1)
		ec, ok := cov.Endpoints["/foos"]
		require.True(t, ok)
		assert.Len(t, ec.Timings, 1)
		assert.Len(t, ec.Unmet, 1)
		assert.Len(t, ec.Skipped, 1)
		require.Len(t, ec.Methods, 1)
		mc, ok := ec.Methods["GET"]
		require.True(t, ok)
		assert.Len(t, ec.Timings, 1)
		assert.Len(t, mc.Unmet, 1)
		assert.Len(t, ec.Skipped, 1)
	})
	t.Run("expectation failure", func(t *testing.T) {
		ctx := newContext(nil)
		ctx.httpDo = &dummyDo{
			status: http.StatusOK,
			body:   []byte(`{"foo":42}`),
		}
		ctx.currEndpoint = Endpoint("/foos", "")
		cov := coverage.NewCoverage()
		ctx.coverage = cov
		m := Method(GET, "").AssertGreaterThan(1, Var("missing"))
		err := m.Run(ctx)
		require.NoError(t, err)
		require.True(t, ctx.failed)
		assert.Len(t, cov.Timings, 1)
		assert.Len(t, cov.Failures, 1)
		require.Len(t, cov.Endpoints, 1)
		ec, ok := cov.Endpoints["/foos"]
		require.True(t, ok)
		assert.Len(t, ec.Timings, 1)
		assert.Len(t, ec.Failures, 1)
		mc, ok := ec.Methods["GET"]
		require.True(t, ok)
		assert.Len(t, ec.Timings, 1)
		assert.Len(t, mc.Failures, 1)
	})
}
