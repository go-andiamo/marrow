package marrow

import (
	"github.com/go-andiamo/urit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_pathParams_resolve(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		pp := pathParams{
			"foo",
			Var("bar"),
		}
		assert.Equal(t, urit.Positions, pp.VarsType())
		ctx := newTestContext(map[Var]any{
			"bar": 42,
		})
		ppr, err := pp.resolve(ctx)
		require.NoError(t, err)
		require.Equal(t, pathParams{"foo", 42}, ppr)
	})
	t.Run("errors", func(t *testing.T) {
		pp := pathParams{
			"foo",
			Var("bar"),
		}
		ctx := newTestContext(nil)
		_, err := pp.resolve(ctx)
		require.Error(t, err)
	})
}

func Test_pathParams_GetPositional(t *testing.T) {
	pp := pathParams{
		"foo",
		Var("bar"),
	}
	assert.Equal(t, 2, pp.Len())
	ctx := newTestContext(map[Var]any{
		"bar": 42,
	})
	ppr, err := pp.resolve(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, ppr.Len())
	v, ok := ppr.GetPositional(0)
	require.True(t, ok)
	require.Equal(t, "foo", v)
	v, ok = ppr.GetPositional(1)
	require.True(t, ok)
	require.Equal(t, "42", v)
	_, ok = ppr.GetPositional(2)
	require.False(t, ok)
}

func Test_pathParams_interface(t *testing.T) {
	pp := pathParams{}
	var pi urit.PathVars = pp
	_, ok := pi.GetNamed("", 0)
	require.False(t, ok)
	_, ok = pi.GetNamedFirst("")
	require.False(t, ok)
	_, ok = pi.GetNamedLast("")
	require.False(t, ok)
	_, ok = pi.Get()
	require.False(t, ok)
	require.Empty(t, pi.GetAll())
	pp.Clear()
	assert.NoError(t, pi.AddNamedValue("", nil))
	assert.NoError(t, pi.AddPositionalValue(nil))
}
