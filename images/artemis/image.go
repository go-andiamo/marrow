package artemis

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	tc "github.com/testcontainers/testcontainers-go/modules/artemis"
	"strings"
)

type image struct {
	options        Options
	mappedPort     string
	stompPort      string
	container      *tc.Container
	client         Client
	topicListeners map[string]*listener
	queueListeners map[string]*listener
}

const (
	defaultBrokerPort = "61616"
	defaultStompPort  = "61613"
	//defaultHTTPPort   = "8161"
)

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
	if i.container, err = tc.Run(ctx, i.options.useImage(), i); err == nil {
		var mp nat.Port
		if mp, err = i.container.MappedPort(ctx, nat.Port(defaultBrokerPort+"/tcp")); err == nil {
			i.mappedPort = mp.Port()
			if mp, err = i.container.MappedPort(ctx, nat.Port(defaultStompPort+"/tcp")); err == nil {
				i.stompPort = mp.Port()
				if err = i.createClient(); err == nil {
					err = i.setupListeners()
				}
			}
		}
	}
	return err
}

func (i *image) createClient() (err error) {
	i.client, err = newClient("localhost:"+i.stompPort, i.options)
	return err
}

func (i *image) shutdown() {
	for _, l := range i.topicListeners {
		l.stop()
	}
	for _, l := range i.queueListeners {
		l.stop()
	}
	i.client.Close()
	if i.container != nil && !i.options.LeaveRunning {
		_ = i.container.Terminate(context.Background())
	}
}

func (i *image) Customize(req *testcontainers.GenericContainerRequest) error {
	req.Env["ARTEMIS_USER"] = i.options.username()
	req.Env["ARTEMIS_PASSWORD"] = i.options.password()
	req.ExposedPorts = append(req.ExposedPorts, defaultStompPort+"/tcp")
	if len(i.options.CreateQueues) > 0 {
		req.Env["EXTRA_ARGS"] = "--http-host 0.0.0.0 --relax-jolokia --queues " + strings.Join(i.options.CreateQueues, ",")
	}
	return nil
}

func (i *image) Container() testcontainers.Container {
	return i.container
}

func (i *image) Client() Client {
	return i.client
}

const ImageName = "artemis"

func (i *image) Name() string {
	return ImageName
}

func (i *image) Host() string {
	return "localhost"
}

func (i *image) Port() string {
	return defaultBrokerPort
}

func (i *image) MappedPort() string {
	return i.mappedPort
}

func (i *image) IsDocker() bool {
	return true
}

func (i *image) Username() string {
	return i.options.username()
}

func (i *image) Password() string {
	return i.options.password()
}
