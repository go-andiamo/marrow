package dragonfly

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/go-redis/redis/v7"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type image struct {
	options        Options
	mappedPort     string
	container      testcontainers.Container
	client         Client
	rClient        *redis.Client
	topicListeners map[string]*listener
	queueListeners map[string]*listener
}

func (i *image) Start() (err error) {
	if i.container != nil {
		return errors.New("already started")
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("start container: %w", err)
		}
	}()
	ctx := context.Background()
	port := i.options.defaultPort()
	natPort := nat.Port(port + "/tcp")
	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        i.options.useImage(),
			ExposedPorts: []string{port},
			WaitingFor:   wait.ForListeningPort(natPort),
		},
		Started: true,
	}
	if i.container, err = testcontainers.GenericContainer(ctx, req); err == nil {
		var ir *container.InspectResponse
		if ir, err = i.container.Inspect(ctx); err == nil {
			if mapped, ok := ir.NetworkSettings.Ports[natPort]; ok {
				i.mappedPort = mapped[0].HostPort
				i.client, i.rClient = newClient(i.Host(), i.mappedPort)
				err = i.setupListeners()
			} else {
				err = fmt.Errorf("could not find port %s in container", port)
			}
		}
	}
	return err
}

func (i *image) shutdown() {
	if i.container != nil && !i.options.LeaveRunning {
		_ = i.container.Terminate(context.Background())
	}
}

func (i *image) MappedPort() string {
	return i.mappedPort
}

func (i *image) Container() testcontainers.Container {
	return i.container
}

const ImageName = "dragonfly"

func (i *image) Name() string {
	return ImageName
}

func (i *image) Host() string {
	return "localhost"
}

func (i *image) Port() string {
	return i.options.defaultPort()
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

func (i *image) Client() Client {
	return i.client
}

func (i *image) RedisClient() *redis.Client {
	return i.rClient
}
