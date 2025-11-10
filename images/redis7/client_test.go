package redis7

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImage_Client(t *testing.T) {
	img := &image{
		options: Options{
			DisableAutoShutdown: true,
		},
	}

	err := img.Start()
	defer func() {
		img.shutdown()
	}()
	require.NoError(t, err)
	c := img.Client()
	require.NotNil(t, c)
	require.NotNil(t, img.RedisClient())
	t.Run("get/set/delete/exists key", func(t *testing.T) {
		_, err = c.Get("foo")
		require.Error(t, err)
		assert.Equal(t, err, NotFound)
		exists, err := c.Exists("foo")
		require.NoError(t, err)
		assert.False(t, exists)
		exists, err = c.Delete("foo")
		require.NoError(t, err)
		assert.False(t, exists)

		err = c.Set("foo", "bar", 0)
		require.NoError(t, err)
		exists, err = c.Exists("foo")
		require.NoError(t, err)
		assert.True(t, exists)
		v, err := c.Get("foo")
		require.NoError(t, err)
		assert.Equal(t, "bar", v)
		exists, err = c.Delete("foo")
		require.NoError(t, err)
		assert.True(t, exists)
	})
	t.Run("publish/subscribe", func(t *testing.T) {
		called := false
		message := ""
		sub := func(msg string) {
			message = msg
			called = true
		}
		closeFn := c.Subscribe("topic_foo", sub)
		defer closeFn()

		err = c.Publish("topic_foo", "bar")
		require.NoError(t, err)
		time.Sleep(time.Second)
		assert.True(t, called)
		assert.Equal(t, "bar", message)

		err = c.Publish("topic_foo", map[string]any{"foo": "bar"})
		require.NoError(t, err)
		time.Sleep(time.Second)
		assert.True(t, called)
		assert.Equal(t, `{"foo":"bar"}`, message)
	})
	t.Run("send/consume", func(t *testing.T) {
		called := false
		message := ""
		sub := func(msg string) {
			message = msg
			called = true
		}
		closeFn := c.Consume("queue_foo", sub)
		defer closeFn()

		err = c.Send("queue_foo", "bar")
		require.NoError(t, err)
		time.Sleep(time.Second)
		assert.True(t, called)
		assert.Equal(t, "bar", message)

		err = c.Send("queue_foo", map[string]any{"foo": "bar"})
		require.NoError(t, err)
		time.Sleep(time.Second)
		assert.True(t, called)
		assert.Equal(t, `{"foo":"bar"}`, message)
	})
	t.Run("queue length", func(t *testing.T) {
		err = c.Send("queue_foo2", "bar")
		require.NoError(t, err)
		time.Sleep(time.Second)
		l, err := c.QueueLength("queue_foo2")
		require.NoError(t, err)
		assert.Equal(t, 1, l)
	})
}
