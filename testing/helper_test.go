package testing

import (
	"bytes"
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewHelper(t *testing.T) {
	var buf bytes.Buffer
	h := NewHelper(nil, &buf, &buf)
	require.NotNil(t, h)
	assert.False(t, h.Failed())
	assert.Equal(t, context.Background(), h.Context())
	raw, ok := h.(*helper)
	require.True(t, ok)
	assert.Nil(t, raw.wrapped)
	assert.Nil(t, raw.parent)
	assert.NotNil(t, raw.frame)
	assert.Equal(t, "tRunner", raw.name)
	assert.False(t, raw.failed)
	assert.False(t, raw.stopped)
	h.End()
	assert.Contains(t, buf.String(), "=== RUN   tRunner")
	assert.Contains(t, buf.String(), "\n--- PASS: tRunner")
}

func TestHelper_Run(t *testing.T) {
	t.Run("pass", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewHelper(nil, &buf, &buf)
		h.Run("subtest", func(t Helper) {})
		h.End()
		assert.False(t, h.Failed())
		assert.Contains(t, buf.String(), "=== RUN   tRunner")
		assert.Contains(t, buf.String(), "\n--- PASS: tRunner (")
		assert.Contains(t, buf.String(), "=== RUN   tRunner/subtest")
		assert.Contains(t, buf.String(), "\n--- PASS: tRunner/subtest (")
	})
	t.Run("pass & log", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewHelper(nil, &buf, &buf)
		h.Run("subtest", func(t Helper) {
			t.Log("something\nsomething else")
		})
		h.End()
		assert.False(t, h.Failed())
		assert.Contains(t, buf.String(), "=== RUN   tRunner")
		assert.Contains(t, buf.String(), "\n--- PASS: tRunner (")
		assert.Contains(t, buf.String(), "=== RUN   tRunner/subtest")
		assert.Contains(t, buf.String(), "\n--- PASS: tRunner/subtest (")
		assert.Contains(t, buf.String(), ": something\n")
		assert.Contains(t, buf.String(), "\n        something else")
	})
	t.Run("fail", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewHelper(nil, &buf, &buf)
		h.Run("subtest", func(t Helper) {
			t.Fail()
		})
		h.End()
		assert.True(t, h.Failed())
		assert.Contains(t, buf.String(), "=== RUN   tRunner")
		assert.Contains(t, buf.String(), "\n--- FAIL: tRunner (")
		assert.Contains(t, buf.String(), "=== RUN   tRunner/subtest")
		assert.Contains(t, buf.String(), "\n--- FAIL: tRunner/subtest (")
	})
	t.Run("fail now", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewHelper(nil, &buf, &buf)
		h.Run("subtest", func(t Helper) {
			t.FailNow()
		})
		h.End()
		assert.True(t, h.Failed())
		assert.Contains(t, buf.String(), "=== RUN   tRunner")
		assert.Contains(t, buf.String(), "\n--- FAIL: tRunner (")
		assert.Contains(t, buf.String(), "=== RUN   tRunner/subtest")
		assert.Contains(t, buf.String(), "\n--- FAIL: tRunner/subtest (")
	})
	t.Run("fatal", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewHelper(nil, &buf, &buf)
		h.Run("subtest", func(t Helper) {
			t.Fatal("something bad happened")
		})
		h.End()
		assert.True(t, h.Failed())
		assert.Contains(t, buf.String(), "=== RUN   tRunner")
		assert.Contains(t, buf.String(), "\n--- FAIL: tRunner (")
		assert.Contains(t, buf.String(), "=== RUN   tRunner/subtest")
		assert.Contains(t, buf.String(), "\n--- FAIL: tRunner/subtest (")
		assert.Contains(t, buf.String(), ": something bad happened")
	})
	t.Run("error", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewHelper(nil, &buf, &buf)
		h.Run("subtest", func(t Helper) {
			t.Error(errors.New("fooey"))
		})
		h.End()
		assert.True(t, h.Failed())
		assert.Contains(t, buf.String(), "=== RUN   tRunner")
		assert.Contains(t, buf.String(), "\n--- FAIL: tRunner (")
		assert.Contains(t, buf.String(), "=== RUN   tRunner/subtest")
		assert.Contains(t, buf.String(), "\n--- FAIL: tRunner/subtest (")
		assert.Contains(t, buf.String(), ": fooey")
	})
	t.Run("stopped", func(t *testing.T) {
		var buf bytes.Buffer
		h := NewHelper(nil, &buf, &buf)
		h.FailNow()
		h.Run("subtest", func(t Helper) {
			t.Error(errors.New("fooey"))
		})
		h.End()
		assert.True(t, h.Failed())
		assert.Contains(t, buf.String(), "=== RUN   tRunner")
		assert.Contains(t, buf.String(), "\n--- FAIL: tRunner (")
		assert.NotContains(t, buf.String(), "=== RUN   tRunner/subtest")
		assert.NotContains(t, buf.String(), "\n--- FAIL: tRunner/subtest (")
		assert.NotContains(t, buf.String(), ": fooey")
	})
	t.Run("wrapped", func(t *testing.T) {
		h := NewHelper(t, nil, nil)
		h.Run("subtest", func(ts Helper) {
			assert.True(t, true)
		})
		assert.False(t, h.Failed())
	})
	t.Run("wrapped, log", func(t *testing.T) {
		h := NewHelper(t, nil, nil)
		h.Run("subtest", func(ts Helper) {
			assert.True(t, true)
			ts.Log("test something")
		})
		assert.False(t, h.Failed())
	})
}
