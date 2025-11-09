package kafka

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImage_Start(t *testing.T) {
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
	assert.NotNil(t, img.Container())
	assert.NotNil(t, img.Client())
}
