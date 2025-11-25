package artemis

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImage_Start(t *testing.T) {
	img := &image{
		options: Options{
			CreateQueues: []string{"foo", "bar"},
		},
	}
	err := img.Start()
	require.NoError(t, err)
	assert.NotNil(t, img.Container())
}
