package localstack

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_dynamoImage(t *testing.T) {
	img := &dynamoImage{
		mappedPort: "123",
		host:       "localhost",
	}
	assert.Equal(t, dynamoImageName, img.Name())
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

func Test_valueToAttributeValue(t *testing.T) {
	testCases := []struct {
		value     any
		expect    any
		expectErr bool
	}{
		{
			value:  nil,
			expect: &types.AttributeValueMemberNULL{Value: true},
		},
		{
			value:  &types.AttributeValueMemberNULL{Value: true},
			expect: &types.AttributeValueMemberNULL{Value: true},
		},
		{
			value: []types.AttributeValue{
				&types.AttributeValueMemberNULL{Value: true},
				&types.AttributeValueMemberS{Value: "foo"},
			},
			expect: &types.AttributeValueMemberL{Value: []types.AttributeValue{
				&types.AttributeValueMemberNULL{Value: true},
				&types.AttributeValueMemberS{Value: "foo"},
			}},
		},
		{
			value:  "foo",
			expect: &types.AttributeValueMemberS{Value: "foo"},
		},
		{
			value:  []string{"foo", "bar"},
			expect: &types.AttributeValueMemberSS{Value: []string{"foo", "bar"}},
		},
		{
			value:  true,
			expect: &types.AttributeValueMemberBOOL{Value: true},
		},
		{
			value:  []byte("foo"),
			expect: &types.AttributeValueMemberB{Value: []byte("foo")},
		},
		{
			value:  [][]byte{[]byte("foo"), []byte("bar")},
			expect: &types.AttributeValueMemberBS{Value: [][]byte{[]byte("foo"), []byte("bar")}},
		},
		{
			value:  float32(1.1),
			expect: &types.AttributeValueMemberN{Value: "1.1"},
		},
		{
			value:  float64(1.1),
			expect: &types.AttributeValueMemberN{Value: "1.1"},
		},
		{
			value:  42,
			expect: &types.AttributeValueMemberN{Value: "42"},
		},
		{
			value:  int64(42),
			expect: &types.AttributeValueMemberN{Value: "42"},
		},
		{
			value: map[string]any{
				"foo": "bar",
				"bar": 42,
			},
			expect: &types.AttributeValueMemberM{Value: map[string]types.AttributeValue{
				"foo": &types.AttributeValueMemberS{Value: "bar"},
				"bar": &types.AttributeValueMemberN{Value: "42"},
			}},
		},
		{
			value:     struct{}{},
			expectErr: true,
		},
		{
			value: map[string]any{
				"foo": struct{}{},
			},
			expectErr: true,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("[%d]", i+1), func(t *testing.T) {
			v, err := valueToAttributeValue(tc.value)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expect, v)
			}
		})
	}
}

var testDynamoOptions = DynamoOptions{
	CreateTables: []dynamodb.CreateTableInput{
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
				{
					AttributeName: aws.String("value"),
					AttributeType: types.ScalarAttributeTypeS,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("code"),
					KeyType:       types.KeyTypeHash,
				},
			},
			GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
				{
					IndexName: aws.String("value-idx"),
					KeySchema: []types.KeySchemaElement{
						{
							AttributeName: aws.String("value"),
							KeyType:       types.KeyTypeHash,
						},
					},
					Projection: &types.Projection{
						ProjectionType: types.ProjectionTypeAll,
					},
				},
			},
		},
	},
}
