package localstack

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOptions_Defaults(t *testing.T) {
	t.Run("version", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultVersion, o.version())
		o = Options{ImageVersion: "1.0.0"}
		assert.Equal(t, "1.0.0", o.version())
	})
	t.Run("image", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultImage, o.image())
		o = Options{Image: "foo"}
		assert.Equal(t, "foo", o.image())
	})
	t.Run("useImage", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultImage+":"+defaultVersion, o.useImage())
		o = Options{Image: "foo", ImageVersion: "1.0.0"}
		assert.Equal(t, "foo:1.0.0", o.useImage())
	})
	t.Run("region", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultRegion, o.region())
		o = Options{Region: "foo"}
		assert.Equal(t, "foo", o.region())
	})
	t.Run("accessKey", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultAccessKey, o.accessKey())
		o = Options{AccessKey: "foo"}
		assert.Equal(t, "foo", o.accessKey())
	})
	t.Run("secretKey", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultSecretKey, o.secretKey())
		o = Options{SecretKey: "foo"}
		assert.Equal(t, "foo", o.secretKey())
	})
	t.Run("sessionToken", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultSessionToken, o.sessionToken())
		o = Options{SessionToken: "foo"}
		assert.Equal(t, "foo", o.sessionToken())
	})
	t.Run("services", func(t *testing.T) {
		o := Options{
			Services: Services{Dynamo, Dynamo, S3, S3, SNS, SNS, SQS, SQS},
		}
		svcs := o.services()
		assert.Len(t, svcs, 4)
		_, ok := svcs[Dynamo]
		assert.True(t, ok)
		_, ok = svcs[S3]
		assert.True(t, ok)
		_, ok = svcs[SNS]
		assert.True(t, ok)
		_, ok = svcs[SQS]
		assert.True(t, ok)
	})
}
