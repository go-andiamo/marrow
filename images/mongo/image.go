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
	"strings"
)

func newImage(options Options) *image {
	return &image{
		options: options,
		watches: make(map[string]*watch),
	}
}

type image struct {
	options    Options
	mappedPort string
	container  *mongodb.MongoDBContainer
	client     *mongo.Client
	watches    map[string]*watch
}

func (i *image) Client() *mongo.Client {
	return i.client
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
	opts := []testcontainers.ContainerCustomizer{
		mongodb.WithUsername(i.options.username()),
		mongodb.WithPassword(i.options.password()),
	}
	if i.options.ReplicaSet != "" {
		opts = append(opts, mongodb.WithReplicaSet(i.options.ReplicaSet))
	}
	if i.container, err = mongodb.Run(ctx, i.options.useImage(), opts...); err == nil {
		var mp nat.Port
		if mp, err = i.container.MappedPort(ctx, natPort); err == nil {
			i.mappedPort = mp.Port()
			if err = i.createClient(ctx); err == nil {
				if err = i.createIndices(ctx); err == nil {
					err = i.createWatches(ctx)
				}
			}
		}
	}
	return err
}

func (i *image) createClient(ctx context.Context) (err error) {
	var ep string
	if ep, err = i.container.ConnectionString(ctx); err == nil {
		if i.options.ReplicaSet != "" {
			ep = ep + "&connect=direct"
		}
		opts := options.Client().ApplyURI(ep)
		if i.client, err = mongo.Connect(opts); err == nil {
			err = i.client.Ping(ctx, nil)
		}
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

func (i *image) createWatches(ctx context.Context) (err error) {
	if len(i.options.Watches) > 0 && i.options.ReplicaSet == "" {
		return errors.New("watches can only be used with replica set")
	}
	for k, v := range i.options.Watches {
		pipeline := mongo.Pipeline{}
		var cs *mongo.ChangeStream
		if k == "" {
			cs, err = i.client.Watch(ctx, pipeline)
		} else if parts := strings.SplitN(k, "/", 2); len(parts) == 2 {
			cs, err = i.client.Database(parts[0]).Collection(parts[1]).Watch(ctx, pipeline)
		} else {
			cs, err = i.client.Database(k).Watch(ctx, pipeline)
		}
		if err != nil {
			return err
		}
		i.watches[k] = newWatch(cs, v.MaxMessages)
	}
	return err
}

func (i *image) shutdown() {
	if i.container != nil && !i.options.LeaveRunning {
		for _, w := range i.watches {
			w.stop()
		}
		_ = i.container.Terminate(context.Background())
	}
}

func (i *image) Container() testcontainers.Container {
	return i.container
}

const ImageName = "mongo"

func (i *image) Name() string {
	return ImageName
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
