package kafka

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
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

	called := 0
	close1 := c.Subscribe("foo", func(message Message) (mark string) {
		called++
		return ""
	})
	close2 := c.Subscribe("foo", func(message Message) (mark string) {
		called++
		return "ok"
	})
	// wait for topic to be created...
	time.Sleep(5 * time.Second)
	err = c.Publish("foo", "bar", "baz")
	require.NoError(t, err)
	err = c.PublishRaw("foo", []byte("bar"), []byte("baz"), Header{
		Key:   []byte("hdr1"),
		Value: []byte("val1"),
	})
	time.Sleep(time.Second)
	assert.Equal(t, 4, called)
	close1()
	close2()
	// create a new subscription that's closed by shutdown...
	c.Subscribe("foo", func(message Message) (mark string) {
		return ""
	})
}
