package kafka

import (
	"github.com/IBM/sarama"
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
	t.Run("defaultPort", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultPort, o.defaultPort())
		o = Options{DefaultPort: "50000"}
		assert.Equal(t, "50000", o.defaultPort())
	})
	t.Run("clusterId", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultClusterId, o.clusterId())
		o = Options{ClusterId: "foo"}
		assert.Equal(t, "foo", o.clusterId())
	})
	t.Run("groupId", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, defaultGroupId, o.groupId())
		o = Options{GroupId: "foo"}
		assert.Equal(t, "foo", o.groupId())
	})
	t.Run("offsetInitial", func(t *testing.T) {
		o := Options{}
		assert.Equal(t, sarama.OffsetNewest, o.offsetInitial())
		o = Options{InitialOffsetOldest: true}
		assert.Equal(t, sarama.OffsetOldest, o.offsetInitial())
	})
}
