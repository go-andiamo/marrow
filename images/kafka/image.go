package kafka

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
	"os"
	"time"
)

type image struct {
	options        Options
	mappedPort     string
	container      *kafka.KafkaContainer
	brokers        []string
	client         Client
	topicListeners map[string]*listener
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
	if i.container, err = kafka.Run(ctx, i.options.useImage(), kafka.WithClusterID(i.options.clusterId())); err == nil {
		var mp nat.Port
		if mp, err = i.container.MappedPort(ctx, natPort); err == nil {
			i.mappedPort = mp.Port()
			if i.brokers, err = i.container.Brokers(ctx); err == nil {
				if len(i.brokers) == 0 {
					err = errors.New("no available brokers")
					return err
				}
				if i.client, err = newClient(i.brokers, i.options); err == nil {
					if err = i.setupListeners(); err == nil && i.options.Wait > 0 {
						time.Sleep(i.options.Wait)
					}
				}
			}
		}
	}
	return err
}

func (i *image) shutdown() {
	if i.client != nil {
		_ = i.client.Close()
		i.client = nil
	}
	if i.container != nil && !i.options.LeaveRunning {
		_ = i.container.Terminate(context.Background())
	}
}

func (i *image) Container() testcontainers.Container {
	return i.container
}

func (i *image) Client() Client {
	return i.client
}

const imageName = "kafka"

func (i *image) Name() string {
	return imageName
}

func (i *image) Host() string {
	return "localhost"
}

func (i *image) Port() string {
	return i.options.defaultPort()
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
