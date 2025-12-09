package localstack

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/docker/go-connections/nat"
	"github.com/go-andiamo/marrow/with"
	"github.com/testcontainers/testcontainers-go"
	tcls "github.com/testcontainers/testcontainers-go/modules/localstack"
	"strings"
	"sync"
)

type image struct {
	options      Options
	container    *tcls.LocalStackContainer
	shuttingDown bool
	services     map[Service]with.Image
	cwlc         *cwl.Client
	mutex        sync.RWMutex
	host         string
	mappedPort   string
}

const (
	defaultPort             = "4566"
	defaultNatPort nat.Port = defaultPort + "/tcp"
)

func (i *image) Start() (err error) {
	i.mutex.Lock()
	defer func() {
		i.mutex.Unlock()
		if err != nil {
			err = fmt.Errorf("start container: %w", err)
		}
	}()
	if i.container != nil || i.shuttingDown {
		return errors.New("already started / shutting down")
	}
	svcs := i.options.services()
	if len(svcs) == 0 && len(i.options.CustomServices) == 0 {
		return errors.New("no services defined")
	}
	i.services = make(map[Service]with.Image, len(svcs))
	ctx := context.Background()
	if i.container, err = tcls.Run(ctx, i.options.useImage(), i); err == nil {
		var mp nat.Port
		if mp, err = i.container.MappedPort(ctx, defaultNatPort); err == nil {
			i.mappedPort = mp.Port()
			var provider *testcontainers.DockerProvider
			if provider, err = testcontainers.NewDockerProvider(); err == nil {
				defer func() {
					_ = provider.Close()
				}()
				if i.host, err = provider.DaemonHost(ctx); err == nil {
					var cfg aws.Config
					if cfg, err = i.buildAwsConfig(ctx); err == nil {
						i.cloudwatchClient(ctx, cfg)
						for svc := range svcs {
							switch svc {
							case Dynamo:
								err = i.createDynamoImage(ctx, cfg)
							case S3:
								err = i.createS3Image(ctx, cfg)
							case SNS:
								err = i.createSnsImage(ctx, cfg)
							case SQS:
								err = i.createSqsImage(ctx, cfg)
							case SecretsManager:
								err = i.createSecretsManagerImage(ctx, cfg)
							case Lambda:
								err = i.createLambdaImage(ctx, cfg)
							case SSM:
								err = i.createSSMImage(ctx, cfg)
							}
							if err != nil {
								return err
							}
						}
						for id, csfn := range i.options.CustomServices {
							if csfn != nil {
								if img, err := csfn(ctx, cfg, i.host, i.mappedPort); err == nil {
									svcId := maxService + Service(id)
									i.services[svcId] = img
								} else {
									return err
								}
							}
						}
					}
				}
			}
		}
	}
	return err
}

func (i *image) cloudwatchClient(ctx context.Context, awsCfg aws.Config) {
	i.cwlc = cwl.NewFromConfig(awsCfg,
		func(o *cwl.Options) {
			o.BaseEndpoint = i.baseEndpoint()
			o.EndpointResolverV2 = cwl.NewDefaultEndpointResolverV2()
		},
	)
}

func (i *image) Customize(req *testcontainers.GenericContainerRequest) error {
	req.Env["LAMBDA_EXECUTOR"] = "local"
	req.Env["LOCALSTACK_LAMBDA_EXECUTOR"] = "local"
	return nil
}

func (i *image) DynamoClient() (client *dynamodb.Client) {
	if svc, ok := i.services[Dynamo]; ok && svc != nil {
		client = svc.(*dynamoImage).client
	}
	return client
}

func (i *image) S3Client() (client *s3.Client) {
	if svc, ok := i.services[S3]; ok && svc != nil {
		client = svc.(*s3Image).client
	}
	return client
}

func (i *image) SNSClient() (client *sns.Client) {
	if svc, ok := i.services[SNS]; ok && svc != nil {
		client = svc.(*snsImage).client
	}
	return client
}

func (i *image) SQSClient() (client *sqs.Client) {
	if svc, ok := i.services[SQS]; ok && svc != nil {
		client = svc.(*sqsImage).client
	}
	return client
}

func (i *image) SecretsManagerClient() (client *secretsmanager.Client) {
	if svc, ok := i.services[SecretsManager]; ok && svc != nil {
		client = svc.(*secretsManagerImage).client
	}
	return client
}

func (i *image) LambdaClient() (client *lambda.Client) {
	if svc, ok := i.services[Lambda]; ok && svc != nil {
		client = svc.(*lambdaImage).client
	}
	return client
}

func (i *image) SSMClient() (client *ssm.Client) {
	if svc, ok := i.services[SSM]; ok && svc != nil {
		client = svc.(*ssmImage).client
	}
	return client
}

func (i *image) buildAwsConfig(ctx context.Context) (aws.Config, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(i.options.region()),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(i.options.accessKey(), i.options.secretKey(), i.options.sessionToken())),
	}
	return config.LoadDefaultConfig(ctx, opts...)
}

func (i *image) baseEndpoint() *string {
	return aws.String(fmt.Sprintf("http://%s:%s", i.host, i.mappedPort))
}

type shutable interface {
	shutdown()
}

func (i *image) shutdown() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if !i.shuttingDown {
		i.shuttingDown = true
		if i.container != nil && !i.options.LeaveRunning {
			_ = i.container.Terminate(context.Background())
		}
		for _, svc := range i.services {
			if ss, ok := svc.(shutable); ok {
				ss.shutdown()
			}
		}
	}
	return
}

func (i *image) Container() testcontainers.Container {
	return i.container
}

const ImageName = "aws"

func (i *image) Name() string {
	return ImageName
}

func (i *image) Host() string {
	return i.host
}

func (i *image) Port() string {
	return defaultPort
}

func (i *image) MappedPort() string {
	return i.mappedPort
}

func (i *image) IsDocker() bool {
	return true
}

func (i *image) Username() string {
	return ""
}

func (i *image) Password() string {
	return ""
}

func (i *image) ResolveEnv(tokens ...string) (string, bool) {
	if len(tokens) > 0 {
		switch strings.ToLower(tokens[0]) {
		case "region":
			return i.options.region(), true
		case "accesskey":
			return i.options.accessKey(), true
		case "secretkey":
			return i.options.secretKey(), true
		case "sessiontoken":
			return i.options.sessionToken(), true
		}
	}
	return "", false
}
