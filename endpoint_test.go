package marrow

import (
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
			SetVar(Before, "foo", nil), SetVar(After, "foo", nil))
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.befores, 1)
		assert.Len(t, raw.afters, 1)
	})
	t.Run("with befores/afters", func(t *testing.T) {
		bas := []BeforeAfter_{
			SetVar(Before, "foo", nil),
			SetVar(After, "foo", nil),
			nil,
		}
		e := Endpoint("/api/foos/{id}", "Foos", bas)
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.befores, 1)
		assert.Len(t, raw.afters, 1)
	})
	t.Run("with mixed", func(t *testing.T) {
		ops := []any{
			SetVar(Before, "foo", nil),
			SetVar(After, "foo", nil),
			nil,
			Method(GET, ""),
		}
		e := Endpoint("/api/foos/{id}", "Foos", ops)
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.methods, 1)
		assert.Len(t, raw.befores, 1)
		assert.Len(t, raw.afters, 1)
	})
	t.Run("panics with unsupported op type", func(t *testing.T) {
		require.Panics(t, func() {
			_ = Endpoint("/api/foos/{id}", "Foos", "not a valid option")
		})
		require.Panics(t, func() {
			_ = Endpoint("/api/foos/{id}", "Foos", []any{"not a valid option"})
		})
	})
	/*
		e := Endpoint("/api/foos/{id}", "Foos",
			DbClearTable(Before, "foo_table"),
			DbClearTable(After, "foo_table"),
			SetVar(Before, "foo_id", "id"),
			Method(GET, "GET foos"),
		)
		require.NotNil(t, e)
		raw, ok := e.(*endpoint)
		require.True(t, ok)
		assert.Len(t, raw.methods, 1)
		assert.Len(t, raw.befores, 2)
		assert.Len(t, raw.afters, 1)
	*/
}
