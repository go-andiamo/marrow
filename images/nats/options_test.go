package nats

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
	t.Run("username", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultUser, o.username())
		o = Options{Username: "foo"}
		assert.Equal(t, "foo", o.username())
	})
	t.Run("password", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultPassword, o.password())
		o = Options{Password: "foo"}
		assert.Equal(t, "foo", o.password())
	})
	t.Run("secret", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultSecret, o.secret())
		o = Options{SecretToken: "foo"}
		assert.Equal(t, "foo", o.secret())
	})
}
