package with

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMake(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		m := Make(Initial, absPath("../_testdata/Makefile.ok"), 0, false)
		assert.Equal(t, Initial, m.Stage())
		assert.Nil(t, m.Shutdown())
	})
	t.Run("panics with relative path", func(t *testing.T) {
		require.Panics(t, func() {
			_ = Make(Initial, "../_testdata/Makefile.ok", 0, false)
		})
	})
	t.Run("panics with bad stage", func(t *testing.T) {
		require.Panics(t, func() {
			_ = Make(Final, "", 0, false)
		})
	})
	t.Run("panics with non-existent file", func(t *testing.T) {
		require.Panics(t, func() {
			_ = Make(Initial, absPath("../_testdata/non-existent"), 0, false)
		})
	})
	t.Run("panics with directory", func(t *testing.T) {
		require.Panics(t, func() {
			_ = Make(Initial, absPath("../_testdata"), 0, false)
		})
	})
}

func TestMake_Init(t *testing.T) {
	t.Run("succeeds", func(t *testing.T) {
		m := Make(Initial, absPath("../_testdata/Makefile.ok"), 10*time.Second, true)
		err := m.Init(nil)
		require.NoError(t, err)
	})
	t.Run("fails", func(t *testing.T) {
		m := Make(Initial, absPath("../_testdata/Makefile.fail"), 0, false)
		err := m.Init(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "make failed: exit status 2")
	})
}

func absPath(path string) string {
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Join(cwd, path)
		}
	}
	return path
}

func Test_resolveMakeProgram(t *testing.T) {
	wasMake := os.Getenv("MAKE")
	defer func() {
		os.Setenv("MAKE", wasMake)
	}()
	s, err := resolveMakeProgram()
	require.NoError(t, err)
	assert.NotEmpty(t, s)
	os.Setenv("MAKE", s)
	s2, err := resolveMakeProgram()
	require.NoError(t, err)
	assert.Equal(t, s, s2)
}
