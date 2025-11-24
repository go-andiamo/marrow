package marrow

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDoAfter(t *testing.T) {
	ba := DoAfter(SetVar("foo", "bar"))
	assert.Equal(t, After, ba.When())
	assert.NotNil(t, ba.Frame())
	ctx := newTestContext(nil)
	err := ba.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, "bar", ctx.vars["foo"])

	assert.Nil(t, DoAfter(nil))
}

func TestDoBefore(t *testing.T) {
	ba := DoBefore(SetVar("foo", "bar"))
	assert.Equal(t, Before, ba.When())
	assert.NotNil(t, ba.Frame())
	ctx := newTestContext(nil)
	err := ba.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, "bar", ctx.vars["foo"])

	assert.Nil(t, DoBefore(nil))
}
