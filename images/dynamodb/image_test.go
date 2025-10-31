package dynamodb

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImage_Start(t *testing.T) {
	img := &image{
		options: Options{
			DisableAutoShutdown: true,
			CreateTables:        testTables,
		},
	}

	err := img.Start()
	defer func() {
		img.shutdown()
	}()
	require.NoError(t, err)
	assert.NotNil(t, img.Client())
}

var testTables = []dynamodb.CreateTableInput{
	{
		TableName: aws.String("TestTable"),
		StreamSpecification: &types.StreamSpecification{
			StreamEnabled:  aws.Bool(true),
			StreamViewType: types.StreamViewTypeNewAndOldImages,
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("code"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("code"),
				KeyType:       types.KeyTypeHash,
			},
		},
	},
}
