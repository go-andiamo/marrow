package nats

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestImage_Start(t *testing.T) {
	img := &image{
		options: Options{
			SecretToken: "s3cr3t",
			CreateKeyValueBuckets: map[string]KeyValueBucket{
				"foo": {
					Description: "foo",
					History:     5,
					TTL:         24 * time.Hour,
				},
			},
		},
	}
	err := img.Start()
	require.NoError(t, err)
	v, ok := img.ResolveEnv("secret")
	assert.True(t, ok)
	assert.Equal(t, "s3cr3t", v)
	v, ok = img.ResolveEnv("conn")
	assert.True(t, ok)
	_, ok = img.ResolveEnv("routingPort")
	assert.True(t, ok)
	_, ok = img.ResolveEnv("monitoringPort")
	assert.True(t, ok)
	_, ok = img.ResolveEnv("foo")
	assert.False(t, ok)
}
