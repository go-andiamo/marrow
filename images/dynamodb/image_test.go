package dynamodb

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	_ "github.com/btnguyen2k/godynamo"
)

func TestImage_Start(t *testing.T) {
	img := &image{
		options: Options{
			CreateTables: testTables,
		},
	}

	err := img.Start()
	defer func() {
		img.shutdown()
	}()
	require.NoError(t, err)
	assert.NotNil(t, img.Client())
	assert.NotNil(t, img.Database())

	dbRows, err := img.db.Query(`LIST TABLES`)
	require.NoError(t, err)
	defer dbRows.Close()
	tables := make([]any, 0)
	for dbRows.Next() {
		var tbl any
		require.NoError(t, dbRows.Scan(&tbl))
		tables = append(tables, tbl)
	}
	assert.Len(t, tables, 1)
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
