package localstack

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_secretsManagerImage(t *testing.T) {
	img := &secretsManagerImage{
		mappedPort: "123",
		host:       "localhost",
	}
	assert.Equal(t, SecretsServiceImageName, img.Name())
	assert.Equal(t, defaultPort, img.Port())
	assert.Equal(t, "localhost", img.Host())
	assert.Equal(t, "123", img.MappedPort())
	assert.True(t, img.IsDocker())
	assert.Equal(t, "", img.Username())
	assert.Equal(t, "", img.Password())
	assert.Nil(t, img.Client())
	s, ok := img.ResolveEnv("Region")
	assert.True(t, ok)
	assert.Equal(t, defaultRegion, s)
	s, ok = img.ResolveEnv("AccessKey")
	assert.True(t, ok)
	assert.Equal(t, defaultAccessKey, s)
	s, ok = img.ResolveEnv("SecretKey")
	assert.True(t, ok)
	assert.Equal(t, defaultSecretKey, s)
	s, ok = img.ResolveEnv("SessionToken")
	assert.True(t, ok)
	assert.Equal(t, defaultSessionToken, s)
	_, ok = img.ResolveEnv("Foo")
	assert.False(t, ok)
}
