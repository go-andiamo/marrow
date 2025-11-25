package artemis

import (
	"encoding/json"
	"github.com/go-stomp/stomp/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestClient(t *testing.T) {
	img := &image{
		options: Options{CreateQueues: []string{"foo"}},
	}
	err := img.Start()
	require.NoError(t, err)
	defer func() {
		img.shutdown()
	}()
	c := img.Client()
	t.Run("send/consume", func(t *testing.T) {
		called := 0
		var message *stomp.Message
		sub := func(msg *stomp.Message) {
			message = msg
			called++
		}
		closeFn, err := c.Consume("foo", sub)
		require.NoError(t, err)
		defer closeFn()

		err = c.Send("foo", "bar", map[string]any{"foo": "bar"})
		require.NoError(t, err)
		time.Sleep(time.Second)
		assert.Equal(t, 1, called)
		assert.NotNil(t, message)

		message = nil
		err = c.Send("foo", map[string]any{"foo": "bar"})
		require.NoError(t, err)
		time.Sleep(time.Second)
		assert.Equal(t, 2, called)
		assert.NotNil(t, message)
	})
	t.Run("publish/subscribe", func(t *testing.T) {
		called := 0
		var message *stomp.Message
		sub := func(msg *stomp.Message) {
			message = msg
			called++
		}
		closeFn, err := c.Subscribe("test-topic", sub)
		require.NoError(t, err)
		defer closeFn()

		err = c.Publish("test-topic", "bar")
		require.NoError(t, err)
		time.Sleep(time.Second)
		assert.Equal(t, 1, called)
		assert.NotNil(t, message)

		message = nil
		err = c.Publish("test-topic", map[string]any{"foo": "bar"})
		require.NoError(t, err)
		time.Sleep(time.Second)
		assert.Equal(t, 2, called)
		assert.NotNil(t, message)
	})
}

func TestClient_encode(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		c := &client{}
		body, ct, err := c.encodeMessage(nil)
		require.NoError(t, err)
		assert.Empty(t, body)
		assert.Empty(t, ct)
	})
	t.Run("string", func(t *testing.T) {
		c := &client{}
		body, ct, err := c.encodeMessage("foo")
		require.NoError(t, err)
		assert.Equal(t, []byte("foo"), body)
		assert.Equal(t, "text/plain", ct)
	})
	t.Run("bytes", func(t *testing.T) {
		c := &client{}
		body, ct, err := c.encodeMessage([]byte("foo"))
		require.NoError(t, err)
		assert.Equal(t, []byte("foo"), body)
		assert.Equal(t, "", ct)
	})
	t.Run("map", func(t *testing.T) {
		c := &client{}
		body, ct, err := c.encodeMessage(map[string]any{"foo": "bar"})
		require.NoError(t, err)
		assert.Equal(t, []byte(`{"foo":"bar"}`), body)
		assert.Equal(t, "application/json", ct)
	})
	t.Run("int", func(t *testing.T) {
		c := &client{}
		body, ct, err := c.encodeMessage(42)
		require.NoError(t, err)
		assert.Equal(t, []byte("42"), body)
		assert.Equal(t, "", ct)
	})
	t.Run("with marshaller", func(t *testing.T) {
		c := &client{
			marshaller: func(msg any) ([]byte, string, error) {
				data, err := json.Marshal(msg)
				return data, "custom", err
			},
		}
		body, ct, err := c.encodeMessage(map[string]any{"foo": "bar"})
		require.NoError(t, err)
		assert.Equal(t, []byte(`{"foo":"bar"}`), body)
		assert.Equal(t, "custom", ct)
	})
}
