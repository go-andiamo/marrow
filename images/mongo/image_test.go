package mongo

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"testing"
)

func TestImage_Start(t *testing.T) {
	img := &image{
		options: Options{
			DisableAutoShutdown: true,
			CreateIndices: IndexOptions{
				"my-db": map[string][]mongo.IndexModel{
					"my-collection": {
						{
							Keys:    bson.D{{Key: "email", Value: 1}},
							Options: options.Index().SetName("uniq_email").SetUnique(true),
						},
					},
				},
			},
		},
	}

	err := img.Start()
	defer func() {
		img.shutdown()
	}()
	require.NoError(t, err)
	assert.NotNil(t, img.Container())
	assert.NotNil(t, img.Client())
}
