package localstack

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-andiamo/marrow/with"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImage(t *testing.T) {
	testName := "foo"
	img := &image{
		options: Options{
			Services: Services{Dynamo, S3, SNS, SQS},
			Dynamo: DynamoOptions{
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
						},
						KeySchema: []types.KeySchemaElement{
							{
								AttributeName: aws.String("code"),
								KeyType:       types.KeyTypeHash,
							},
						},
					},
				},
			},
			S3: S3Options{
				CreateBuckets: []s3.CreateBucketInput{
					{
						Bucket: &testName,
					},
				},
			},
			SNS: SNSOptions{
				CreateTopics: []sns.CreateTopicInput{
					{
						Name: &testName,
					},
				},
			},
			SQS: SQSOptions{
				CreateQueues: []sqs.CreateQueueInput{
					{
						QueueName: &testName,
					},
				},
			},
		},
	}
	err := img.Start()
	require.NoError(t, err)
	_, ok := img.services[Dynamo]
	require.True(t, ok)
	_, ok = img.services[S3]
	require.True(t, ok)
	svc, ok := img.services[SNS]
	require.True(t, ok)
	require.NotNil(t, svc)
	s, ok := svc.(with.ImageResolveEnv).ResolveEnv("arn", testName)
	assert.True(t, ok)
	assert.Contains(t, s, ":"+testName)
	assert.Contains(t, s, "arn:aws:sns:us-east-1:")
	svc, ok = img.services[SQS]
	require.True(t, ok)
	require.NotNil(t, svc)
	s, ok = svc.(with.ImageResolveEnv).ResolveEnv("arn", testName)
	assert.True(t, ok)
	assert.Contains(t, s, "/"+testName)
	assert.Contains(t, s, "http://sqs.us-east-1.")

	assert.NotNil(t, img.DynamoClient())
	assert.NotNil(t, img.S3Client())
	assert.NotNil(t, img.SNSClient())
	assert.NotNil(t, img.SQSClient())
}
