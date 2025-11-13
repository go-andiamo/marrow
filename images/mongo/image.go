package mongo

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"os"
)

type image struct {
	options    Options
	mappedPort string
	container  *mongodb.MongoDBContainer
	client     *mongo.Client
}

func (i *image) Client() *mongo.Client {
	return i.client
}

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
	opts := []testcontainers.ContainerCustomizer{
		mongodb.WithUsername(i.options.username()),
		mongodb.WithPassword(i.options.password()),
	}
	if i.container, err = mongodb.Run(ctx, i.options.useImage(), opts...); err == nil {
		var mp nat.Port
		if mp, err = i.container.MappedPort(ctx, natPort); err == nil {
			i.mappedPort = mp.Port()
			if err = i.createClient(ctx); err == nil {
				err = i.createIndices(ctx)
			}
		}
	}
	return err
}

func (i *image) createClient(ctx context.Context) (err error) {
	opts := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%s@%s:%s", i.options.username(), i.options.password(), i.Host(), i.MappedPort()))
	if i.client, err = mongo.Connect(opts); err == nil {
		err = i.client.Ping(ctx, nil)
	}
	return err
}

func (i *image) createIndices(ctx context.Context) (err error) {
	for dbName, collections := range i.options.CreateIndices {
		db := i.client.Database(dbName)
		for collName, indices := range collections {
			if err = db.CreateCollection(ctx, collName); err == nil {
				_, err = db.Collection(collName).Indexes().CreateMany(ctx, indices)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (i *image) shutdown() {
	if i.container != nil && !i.options.LeaveRunning {
		_ = i.container.Terminate(context.Background())
	}
}

func (i *image) Container() testcontainers.Container {
	return i.container
}

const (
	imageName      = "mongo"
	envRyukDisable = "TESTCONTAINERS_RYUK_DISABLED"
)

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
	return i.options.username()
}

func (i *image) Password() string {
	return i.options.password()
}
