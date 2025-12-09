package nats

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/go-connections/nat"
	nc "github.com/nats-io/nats.go"
	"github.com/testcontainers/testcontainers-go"
	tc "github.com/testcontainers/testcontainers-go/modules/nats"
	"strings"
)

type image struct {
	options        Options
	mappedPort     string
	routingPort    string
	monitoringPort string
	container      *tc.NATSContainer
	client         *nc.Conn
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
	opts := []testcontainers.ContainerCustomizer{
		tc.WithUsername(i.options.username()),
		tc.WithPassword(i.options.password()),
		i.secretOpt(),
	}
	if i.container, err = tc.Run(ctx, i.options.useImage(), opts...); err == nil {
		if err = i.mapPorts(ctx); err == nil {
			if err = i.createClient(ctx); err == nil {
				err = i.createBuckets()
			}
		}
	}
	return err
}

func (i *image) secretOpt() testcontainers.ContainerCustomizer {
	const confTemplate = `
listen: 0.0.0.0:4222
authorization {
    token: "%s"
}
`
	return tc.WithConfigFile(strings.NewReader(fmt.Sprintf(confTemplate, i.options.secret())))
}

const (
	defaultClientPort     = "4222"
	defaultRoutingPort    = "6222"
	defaultMonitoringPort = "8222"
)

func (i *image) mapPorts(ctx context.Context) (err error) {
	var mp nat.Port
	if mp, err = i.container.MappedPort(ctx, defaultClientPort+"/tcp"); err == nil {
		i.mappedPort = mp.Port()
		if mp, err = i.container.MappedPort(ctx, nat.Port(defaultRoutingPort+"/tcp")); err == nil {
			i.routingPort = mp.Port()
			if mp, err = i.container.MappedPort(ctx, nat.Port(defaultMonitoringPort+"/tcp")); err == nil {
				i.monitoringPort = mp.Port()
			}
		}
	}
	return err
}

func (i *image) createClient(ctx context.Context) (err error) {
	var uri string
	if uri, err = i.container.ConnectionString(ctx); err == nil {
		i.client, err = nc.Connect(uri, nc.Token(i.options.secret()))
	}
	return err
}

func (i *image) createBuckets() (err error) {
	if len(i.options.CreateKeyValueBuckets) > 0 {
		var js nc.JetStreamContext
		if js, err = i.client.JetStream(); err == nil {
			for k, v := range i.options.CreateKeyValueBuckets {
				if _, err = js.CreateKeyValue(&nc.KeyValueConfig{
					Bucket:       k,
					Description:  v.Description,
					MaxValueSize: v.MaxValueSize,
					History:      v.History,
					TTL:          v.TTL,
					MaxBytes:     v.MaxBytes,
					Storage:      nc.StorageType(v.Storage),
					Compression:  v.Compression,
				}); err != nil {
					return err
				}
			}
		}
	}
	return err
}

func (i *image) shutdown() {
	i.client.Close()
	if i.container != nil && !i.options.LeaveRunning {
		_ = i.container.Terminate(context.Background())
	}
}

func (i *image) Container() testcontainers.Container {
	return i.container
}

func (i *image) Client() *nc.Conn {
	return i.client
}

const ImageName = "nats"

func (i *image) Name() string {
	return ImageName
}

func (i *image) Host() string {
	return "localhost"
}

func (i *image) Port() string {
	return defaultClientPort
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

func (i *image) ResolveEnv(tokens ...string) (string, bool) {
	if len(tokens) > 0 {
		switch strings.ToLower(tokens[0]) {
		case "conn":
			if v, err := i.container.ConnectionString(context.Background()); err == nil {
				return v, true
			}
		case "secret":
			return i.options.SecretToken, true
		case "routingport":
			return i.routingPort, true
		case "monitoringport":
			return i.monitoringPort, true
		}
	}
	return "", false
}
