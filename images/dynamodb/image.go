package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/localstack"
	"os"
)

type image struct {
	options    Options
	mappedPort string
	container  testcontainers.Container
	client     *dynamodb.Client
}

const envRyukDisable = "TESTCONTAINERS_RYUK_DISABLED"

func (i *image) Start() (err error) {
	if i.container != nil {
		return errors.New("already started")
	}
	defer func() {
		_ = os.Setenv(envRyukDisable, "false")
		if err != nil {
			err = fmt.Errorf("start container: %w", err)
		}
	}()
	if i.options.DisableAutoShutdown {
		_ = os.Setenv(envRyukDisable, "true")
	}
	ctx := context.Background()
	port := i.options.defaultPort()
	natPort := nat.Port(port + "/tcp")
	var ls *localstack.LocalStackContainer
	if ls, err = localstack.Run(ctx, i.options.useImage(), &customizer{}); err == nil {
		i.container = ls.Container
		var ir *container.InspectResponse
		if ir, err = i.container.Inspect(ctx); err == nil {
			if mapped, ok := ir.NetworkSettings.Ports[natPort]; ok {
				i.mappedPort = mapped[0].HostPort
				err = i.setupClient(ctx, mapped[0])
			} else {
				err = fmt.Errorf("could not find port %s in container", port)
			}
		}
	}
	return err
}

func (i *image) setupClient(ctx context.Context, mapped nat.PortBinding) (err error) {
	var provider *testcontainers.DockerProvider
	if provider, err = testcontainers.NewDockerProvider(); err == nil {
		var host string
		if host, err = provider.DaemonHost(ctx); err == nil {
			customResolver := aws.EndpointResolverWithOptionsFunc(
				func(service, region string, opts ...interface{}) (aws.Endpoint, error) {
					return aws.Endpoint{
						PartitionID:   "aws",
						URL:           fmt.Sprintf("http://%s:%s", host, mapped.HostPort),
						SigningRegion: region,
					}, nil
				})
			var cfg aws.Config
			region := "us-east-1"
			if cfg, err = awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(region),
				awsConfig.WithEndpointResolverWithOptions(customResolver)); err == nil {
				i.client = dynamodb.NewFromConfig(cfg)
				err = i.createTables(ctx)
			}
		}
	}
	return err
}

func (i *image) createTables(ctx context.Context) error {
	for _, ct := range i.options.CreateTables {
		if ct.BillingMode == "" && ct.ProvisionedThroughput == nil {
			ct.BillingMode = types.BillingModePayPerRequest
		}
		if _, err := i.client.CreateTable(ctx, &ct); err != nil {
			return err
		}
	}
	return nil
}

type customizer struct{}

var _ testcontainers.ContainerCustomizer = (*customizer)(nil)

func (c *customizer) Customize(req *testcontainers.GenericContainerRequest) error {
	//TODO any request customizations?
	return nil
}

func (i *image) shutdown() {
	if i.container != nil {
		_ = i.container.Terminate(context.Background())
	}
}

func (i *image) MappedPort() string {
	return i.mappedPort
}

func (i *image) Container() testcontainers.Container {
	return i.container
}

func (i *image) Client() *dynamodb.Client {
	return i.client
}
