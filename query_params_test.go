package marrow

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestQueryParams_encode(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		qps := make(queryParams, 0)
		enc, err := qps.encode(nil)
		require.NoError(t, err)
		assert.Equal(t, "", enc)
	})
	t.Run("string", func(t *testing.T) {
		qps := make(queryParams, 0)
		qps.add("foo", "x?y&z")
		enc, err := qps.encode(nil)
		require.NoError(t, err)
		assert.Equal(t, `?foo=x%3Fy%26z`, enc)
	})
	t.Run("int", func(t *testing.T) {
		qps := make(queryParams, 0)
		qps.add("foo", 1)
		enc, err := qps.encode(nil)
		require.NoError(t, err)
		assert.Equal(t, `?foo=1`, enc)
	})
	t.Run("multi", func(t *testing.T) {
		qps := make(queryParams, 0)
		qps.add("foo", "x")
		qps.add("foo", "y")
		qps.add("foo", nil)
		qps.add("bar", "z")
		enc, err := qps.encode(nil)
		require.NoError(t, err)
		assert.Equal(t, "?bar=z&foo=x&foo=y&foo", enc)
	})
	t.Run("var", func(t *testing.T) {
		qps := make(queryParams, 0)
		qps.add("foo", Var("bar"))
		enc, err := qps.encode(newContext(map[Var]any{"bar": 42}))
		require.NoError(t, err)
		assert.Equal(t, `?foo=42`, enc)
	})
	t.Run("missing var", func(t *testing.T) {
		qps := make(queryParams, 0)
		qps.add("foo", Var("bar"))
		_, err := qps.encode(newContext(nil))
		require.Error(t, err)
	})
	t.Run("no value", func(t *testing.T) {
		qps := make(queryParams, 0)
		qps.add("foo", nil)
		enc, err := qps.encode(nil)
		require.NoError(t, err)
		assert.Equal(t, `?foo`, enc)
	})
}
