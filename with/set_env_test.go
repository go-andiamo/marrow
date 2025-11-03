package with

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestSetEnv(t *testing.T) {
	t.Run("new", func(t *testing.T) {
		w := SetEnv("foo", "bar")
		assert.Equal(t, Initial, w.Stage())
		assert.Nil(t, w.Shutdown())
	})
	t.Run("init", func(t *testing.T) {
		defer func() {
			_ = os.Setenv("foo", "")
		}()
		w := SetEnv("foo", "bar")
		err := w.Init(nil)
		require.NoError(t, err)
		assert.Equal(t, "bar", os.Getenv("foo"))
	})
}
