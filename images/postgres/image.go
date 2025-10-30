package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"os"
)

type image struct {
	options    Options
	db         *sql.DB
	mappedPort string
	container  testcontainers.Container
}

func (i *image) Start() (err error) {
	if i.container != nil {
		return errors.New("already started")
	}
	if err = i.startContainer(); err == nil {
		if err = i.openDatabase(); err == nil {
			err = i.migrateDatabase()
		}
	}
	return err
}

const envRyukDisable = "TESTCONTAINERS_RYUK_DISABLED"

func (i *image) openDatabase() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("open database: %w", err)
			if i.db != nil {
				_ = i.db.Close()
			}
		}
	}()
	if i.db, err = sql.Open("postgres", i.dsn("localhost", i.options.Database)); err == nil {
		err = i.db.Ping()
	}
	return err
}

func (i *image) startContainer() (err error) {
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
	dbPort := i.options.defaultPort()
	natPort := nat.Port(dbPort + "/tcp")
	req := testcontainers.ContainerRequest{
		Image:        i.options.useImage(),
		ExposedPorts: []string{dbPort + "/tcp"},
		Cmd:          []string{"postgres", "-c", "fsync=off"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort(natPort)),
		Env: map[string]string{
			"POSTGRES_USER":     i.options.username(),
			"POSTGRES_PASSWORD": i.options.password(),
			"POSTGRES_DB":       i.options.Database,
		},
	}
	if i.container, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{ContainerRequest: req, Started: true}); err == nil {
		var ir *container.InspectResponse
		if ir, err = i.container.Inspect(ctx); err == nil {
			if mapped, ok := ir.NetworkSettings.Ports[natPort]; ok {
				i.mappedPort = mapped[0].HostPort
			} else {
				err = fmt.Errorf("could not find port %s in container", dbPort)
			}
		}
	}
	return err
}

func (i *image) dsn(host string, dbName string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		i.options.username(), i.options.password(), host, i.mappedPort, dbName)
}

func (i *image) shutdown() {
	if i.db != nil {
		_ = i.db.Close()
	}
	if i.container != nil {
		_ = i.container.Terminate(context.Background())
	}
}

func (i *image) MappedPort() string {
	return i.mappedPort
}

func (i *image) Database() *sql.DB {
	return i.db
}

func (i *image) Container() testcontainers.Container {
	return i.container
}
