package localstack

import (
	"fmt"
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
}

func TestOptions_services(t *testing.T) {
	testCases := []struct {
		services Services
		expected Services
	}{
		{
			services: Services{Dynamo, S3},
			expected: Services{Dynamo, S3},
		},
		{
			services: Services{Dynamo, S3, S3, -S3},
			expected: Services{Dynamo},
		},
		{
			services: Services{Dynamo, S3, -S3, S3},
			expected: Services{Dynamo, S3},
		},
		{
			services: Services{All},
			expected: Services{Dynamo, S3, SNS, SQS, SecretsManager, Lambda},
		},
		{
			services: Services{All, Except, S3},
			expected: Services{Dynamo, SNS, SQS, SecretsManager, Lambda},
		},
		{
			services: Services{All, -S3},
			expected: Services{Dynamo, SNS, SQS, SecretsManager, Lambda},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			o := Options{
				Services: tc.services,
			}
			reqd := o.services()
			assert.Equal(t, len(tc.expected), len(reqd))
			for _, s := range tc.expected {
				_, ok := reqd[s]
				assert.True(t, ok)
			}
		})
	}
}
