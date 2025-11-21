package dragonfly

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImage_Start(t *testing.T) {
	img := &image{
		options: Options{},
	}

	err := img.Start()
	defer func() {
		img.shutdown()
	}()
	require.NoError(t, err)
}
