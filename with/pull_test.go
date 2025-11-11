package with

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPullImage(t *testing.T) {
	pOptions := PullOptions{
		//Username: "<username>",
		//Password: "<password|pat>",
	}
	rOptions := &RunOptions{
		Name: "redis",
		Port: 6379,
		Env: map[string]any{
			"FOO": "{$foo}",
		},
		LeaveRunning: true,
	}
	pi := PullImage(Supporting, "redis:8.2.3", pOptions, rOptions)

	si := newMockInit()
	err := pi.Init(si)
	require.NoError(t, err)
	assert.Len(t, si.called, 1)
	_, ok := si.called["AddSupportingImage:redis"]
	assert.True(t, ok)
	_, ok = si.images["redis"]
	assert.True(t, ok)

	assert.Equal(t, Supporting, pi.Stage())
	assert.NotNil(t, pi.Shutdown())
	assert.Equal(t, "redis", pi.Name())
	assert.Equal(t, "localhost", pi.Host())
	assert.Equal(t, "6379", pi.Port())
	assert.NotEqual(t, "", pi.MappedPort())
	assert.True(t, pi.IsDocker())
	assert.Equal(t, "", pi.Username())
	assert.Equal(t, "", pi.Password())
	assert.NotNil(t, pi.Container())
	pi.Shutdown()()

	// bad credentials...
	pOptions = PullOptions{
		Username: "<username>",
		Password: "<password|pat>",
	}
	pi = PullImage(Supporting, "redis:8.2.3", pOptions, nil)
	err = pi.Init(si)
	require.Error(t, err)

	require.Panics(t, func() {
		_ = PullImage(Final, "redis:8.2.3", pOptions, nil)
	})
}
