package localstack

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-andiamo/marrow/with"
	"strings"
)

type SQSService interface {
	Client() *sqs.Client
}

type sqsImage struct {
	options    Options
	host       string
	mappedPort string
	client     *sqs.Client
	arns       map[string]string
}

var _ with.Image = (*sqsImage)(nil)
var _ with.ImageResolveEnv = (*sqsImage)(nil)
var _ SQSService = (*sqsImage)(nil)

func (i *image) createSqsImage(ctx context.Context, awsCfg aws.Config) (err error) {
	img := &sqsImage{
		options:    i.options,
		host:       i.host,
		mappedPort: i.mappedPort,
		client: sqs.NewFromConfig(awsCfg,
			func(o *sqs.Options) {
				o.BaseEndpoint = i.baseEndpoint()
			},
		),
		arns: make(map[string]string),
	}
	err = img.createQueues(ctx)
	if err == nil {
		i.services[SQS] = img
	}
	return err
}

func (s *sqsImage) createQueues(ctx context.Context) error {
	for _, queue := range s.options.SQS.CreateQueues {
		if out, err := s.client.CreateQueue(ctx, &queue); err == nil {
			s.arns[*queue.QueueName] = *out.QueueUrl
		} else {
			return err
		}
	}
	return nil
}

func (s *sqsImage) Client() *sqs.Client {
	return s.client
}

const sqsImageName = "sqs"

func (s *sqsImage) Name() string {
	return sqsImageName
}

func (s *sqsImage) Host() string {
	return s.host
}

func (s *sqsImage) Port() string {
	return defaultPort
}

func (s *sqsImage) MappedPort() string {
	return s.mappedPort
}

func (s *sqsImage) IsDocker() bool {
	return true
}

func (s *sqsImage) Username() string {
	return ""
}

func (s *sqsImage) Password() string {
	return ""
}

func (s *sqsImage) ResolveEnv(tokens ...string) (string, bool) {
	if len(tokens) > 0 {
		switch strings.ToLower(tokens[0]) {
		case "region":
			return s.options.region(), true
		case "accesskey":
			return s.options.accessKey(), true
		case "secretkey":
			return s.options.secretKey(), true
		case "sessiontoken":
			return s.options.sessionToken(), true
		case "arn":
			if len(tokens) > 1 {
				v, ok := s.arns[tokens[1]]
				return v, ok
			}
		}
	}
	return "", false
}
