package marrow

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestMethod(t *testing.T) {
	m := Method(GET, "foo")
	m.ExpectOK()
	raw := m.(*method)
	assert.Len(t, raw.expectations, 1)
	exp := raw.expectations[0]
	assert.Equal(t, "Expect OK", exp.Name())
	f := exp.Frame()
	assert.NotNil(t, f)
	assert.Equal(t, t.Name(), f.Name)
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
		req, err := raw.buildRequest(ctx)
		require.NoError(t, err)
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
		_, err := raw.buildRequest(ctx)
		require.Error(t, err)
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

/*
func TestFoo(t *testing.T) {
	Endpoint("/foos", "Test Foos",
		DbClearTable(Before, "foo_table"),
		Method(POST, "Add foo").
			RequestBody(JSON{"foo": "bar"}).ExpectOK().
			ExpectEqual(1, Query("SELECT COUNT(*) FROM foo_table")),
	)

	Endpoint("/foos", "Test Foos",
		Method(POST, "Add foo").
			SetVar(Before, "count", Query("SELECT COUNT(*) FROM foo_table")).
			RequestBody(JSON{"foo": "bar"}).ExpectOK().
			ExpectGreaterThan(Query("SELECT COUNT(*) FROM foo_table"), Var("count")),
	)

	Endpoint("/foos", "Test Foos",
		Method(POST, "Add foo").
			SetVar(Before, "foo_id", "123").
			RequestBody(JSON{"foo": Var("foo_id")}).ExpectOK())
}
*/

func Test_normalizeBody(t *testing.T) {
	var body any
	data := []byte(`[{"foo":42},{"bar":2.2}]`)
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	err := d.Decode(&body)
	require.NoError(t, err)
	normalizedBody, err := normalizeBody(body)
	require.NoError(t, err)
	assert.NotNil(t, normalizedBody)
	assert.Equal(t, int64(42), normalizedBody.([]any)[0].(map[string]any)["foo"])
	assert.Equal(t, 2.2, normalizedBody.([]any)[1].(map[string]any)["bar"])
}
