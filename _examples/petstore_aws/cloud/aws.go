package cloud

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"strings"
)

type MessagePublisher interface {
	PublishMessage(ctx context.Context, topicArn string, msg string) error
}

type AwsClients struct {
	S3             *s3.Client
	SNS            *sns.Client
	SQS            *sqs.Client
	SecretsManager *secretsmanager.Client
	SSM            *ssm.Client
	ParamPrefix    string
}

var _ MessagePublisher = (*AwsClients)(nil)

func NewAwsClients(ctx context.Context, paramPrefix string) (*AwsClients, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &AwsClients{
		S3:             s3.NewFromConfig(cfg),
		SNS:            sns.NewFromConfig(cfg),
		SQS:            sqs.NewFromConfig(cfg),
		SecretsManager: secretsmanager.NewFromConfig(cfg),
		SSM:            ssm.NewFromConfig(cfg),
		ParamPrefix:    paramPrefix,
	}, nil
}

func (c *AwsClients) GetParameter(ctx context.Context, name string) (string, error) {
	if c.ParamPrefix != "" {
		name = strings.TrimSuffix(c.ParamPrefix, "/") + "/" + name
	}
	inp := &ssm.GetParameterInput{
		Name: aws.String(name),
	}
	if out, err := c.SSM.GetParameter(ctx, inp); err == nil {
		return *out.Parameter.Value, nil
	} else {
		return "", fmt.Errorf("aws GetParameter %q: %w", name, err)
	}
}

func (c *AwsClients) GetSecret(ctx context.Context, name string) (string, error) {
	inp := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(name),
	}
	if out, err := c.SecretsManager.GetSecretValue(ctx, inp); err == nil && out.SecretString != nil {
		return *out.SecretString, nil
	} else {
		return "", fmt.Errorf("aws GetSecret %q: %w", name, err)
	}
}

func (c *AwsClients) PublishMessage(ctx context.Context, topicArn string, msg string) error {
	inp := &sns.PublishInput{
		TopicArn: aws.String(topicArn),
		Message:  aws.String(msg),
	}
	_, err := c.SNS.Publish(ctx, inp)
	return err
}
