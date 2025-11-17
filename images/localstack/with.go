package localstack

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-andiamo/marrow/with"
	"github.com/testcontainers/testcontainers-go"
)

type Image interface {
	with.With
	Start() error
	Container() testcontainers.Container
	DynamoClient() *dynamodb.Client
	S3Client() *s3.Client
	SNSClient() *sns.Client
	SQSClient() *sqs.Client
	SecretsManagerClient() *secretsmanager.Client
}

// With creates a new localstack support image for use in marrow.Suite .Init()
func With(options Options) Image {
	return &image{options: options}
}

var _ with.With = (*image)(nil)

func (i *image) Init(init with.SuiteInit) error {
	if err := i.Start(); err != nil {
		return fmt.Errorf("with localstack image init error: %w", err)
	}
	for _, si := range i.services {
		init.AddSupportingImage(si)
	}
	return nil
}

func (i *image) Stage() with.Stage {
	return with.Supporting
}

func (i *image) Shutdown() func() {
	return func() {
		i.shutdown()
	}
}
