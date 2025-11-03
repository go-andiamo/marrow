package mysql

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
		assert.Equal(t, defaultRootUsername, o.rootUsername())
		o = Options{RootUsername: "foo"}
		assert.Equal(t, "foo", o.rootUsername())
	})
	t.Run("password", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultRootPassword, o.rootPassword())
		o = Options{RootPassword: "foo"}
		assert.Equal(t, "foo", o.rootPassword())
	})
	t.Run("defaultPort", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultPort, o.defaultPort())
		o = Options{DefaultPort: "50000"}
		assert.Equal(t, "50000", o.defaultPort())
	})
}

func TestMigration_path(t *testing.T) {
	m := Migration{}
	assert.Equal(t, ".", m.path())
	m = Migration{Path: "foo"}
	assert.Equal(t, "foo", m.path())
}
