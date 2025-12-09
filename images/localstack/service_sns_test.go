package localstack

import (
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_snsImage(t *testing.T) {
	img := &snsImage{
		mappedPort: "123",
		host:       "localhost",
	}
	assert.Equal(t, SNSImageName, img.Name())
	assert.Equal(t, defaultPort, img.Port())
	assert.Equal(t, "localhost", img.Host())
	assert.Equal(t, "123", img.MappedPort())
	assert.True(t, img.IsDocker())
	assert.Equal(t, "", img.Username())
	assert.Equal(t, "", img.Password())
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

func TestSNSPublish(t *testing.T) {
	c := SNSPublish(marrow.After, "my-topic", "foo")
	assert.Equal(t, marrow.After, c.When())
	assert.NotNil(t, c.Frame())
}

func TestSNSMessagesCount(t *testing.T) {
	c := SNSMessagesCount("my-topic", "foo")
	assert.Equal(t, "sns.SNSMessagesCount(\"my-topic\")", fmt.Sprintf("%s", c))
}

func TestSNSMessages(t *testing.T) {
	c := SNSMessages("my-topic")
	assert.Equal(t, "sns.SNSMessages(\"my-topic\")", fmt.Sprintf("%s", c))
}
