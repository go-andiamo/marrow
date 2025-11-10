package localstack

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-andiamo/marrow/with"
	"strings"
)

type dynamoImage struct {
	options    Options
	host       string
	mappedPort string
	client     *dynamodb.Client
}

var _ with.Image = (*dynamoImage)(nil)
var _ with.ImageResolveEnv = (*dynamoImage)(nil)

func (i *image) createDynamoImage(ctx context.Context, awsCfg aws.Config) (err error) {
	img := &dynamoImage{
		options:    i.options,
		host:       i.host,
		mappedPort: i.mappedPort,
		client: dynamodb.NewFromConfig(awsCfg,
			func(o *dynamodb.Options) {
				o.BaseEndpoint = i.baseEndpoint()
				o.EndpointResolverV2 = dynamodb.NewDefaultEndpointResolverV2()
			},
		),
	}
	err = img.createTables(ctx)
	if err == nil {
		i.services[Dynamo] = img
	}
	return err
}

func (s *dynamoImage) createTables(ctx context.Context) error {
	for _, ct := range s.options.Dynamo.CreateTables {
		if ct.BillingMode == "" && ct.ProvisionedThroughput == nil {
			ct.BillingMode = types.BillingModePayPerRequest
		}
		if _, err := s.client.CreateTable(ctx, &ct); err != nil {
			return err
		}
	}
	return nil
}

const dynamoImageName = "dynamo"

func (s *dynamoImage) Name() string {
	return dynamoImageName
}

func (s *dynamoImage) Host() string {
	return s.host
}

func (s *dynamoImage) Port() string {
	return defaultPort
}

func (s *dynamoImage) MappedPort() string {
	return s.mappedPort
}

func (s *dynamoImage) IsDocker() bool {
	return true
}

func (s *dynamoImage) Username() string {
	return ""
}

func (s *dynamoImage) Password() string {
	return ""
}

func (s *dynamoImage) ResolveEnv(tokens ...string) (string, bool) {
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
		}
	}
	return "", false
}
