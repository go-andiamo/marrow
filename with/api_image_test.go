package with

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestApiImage(t *testing.T) {
	i := ApiImage("foo", "bar", 8080, nil, false)
	assert.Nil(t, i.Container())
	assert.Equal(t, "localhost", i.Host())
	assert.Equal(t, "8080", i.Port())
	assert.Equal(t, "", i.MappedPort())
	assert.True(t, i.IsDocker())
	assert.Equal(t, "", i.Username())
	assert.Equal(t, "", i.Password())
	assert.Equal(t, Final, i.Stage())
	assert.NotNil(t, i.Shutdown())
	assert.Equal(t, "foo:bar", i.Name())
}
