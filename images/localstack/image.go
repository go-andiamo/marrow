package localstack

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/docker/go-connections/nat"
	"github.com/go-andiamo/marrow/with"
	"github.com/testcontainers/testcontainers-go"
	tcls "github.com/testcontainers/testcontainers-go/modules/localstack"
	"os"
	"sync"
)

type image struct {
	options      Options
	container    *tcls.LocalStackContainer
	shuttingDown bool
	services     map[Service]with.Image
	mutex        sync.RWMutex
	host         string
	mappedPort   string
}

const (
	defaultPort             = "4566"
	defaultNatPort nat.Port = defaultPort + "/tcp"
	envRyukDisable          = "TESTCONTAINERS_RYUK_DISABLED"
)

func (i *image) Start() (err error) {
	i.mutex.Lock()
	defer func() {
		i.mutex.Unlock()
		_ = os.Setenv(envRyukDisable, "false")
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
	if i.options.DisableAutoShutdown {
		_ = os.Setenv(envRyukDisable, "true")
	}
	if i.container, err = tcls.Run(ctx, i.options.useImage()); err == nil {
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

func (i *image) shutdown() {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if !i.shuttingDown {
		i.shuttingDown = true
		if i.container != nil && !i.options.LeaveRunning {
			_ = i.container.Terminate(context.Background())
		}
	}
	return
}

func (i *image) Container() testcontainers.Container {
	return i.container
}
