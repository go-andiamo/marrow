package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	_ "github.com/go-sql-driver/mysql"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type image struct {
	name       string
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

func (i *image) openDatabase() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("open database: %w", err)
			if i.db != nil {
				_ = i.db.Close()
			}
		}
	}()
	var db *sql.DB
	if db, err = sql.Open("mysql", i.dsn("localhost", "")); err == nil {
		if err = db.Ping(); err == nil {
			if i.options.Database != "" {
				if _, err = db.Exec(fmt.Sprintf("CREATE SCHEMA %s", i.options.Database)); err == nil {
					defer func() {
						_ = db.Close()
					}()
					if i.db, err = sql.Open("mysql", i.dsn("localhost", i.options.Database)); err == nil {
						err = i.db.Ping()
					}
				}
			} else {
				i.db = db
			}
		}
	}
	return err
}

func (i *image) dsn(host string, dbName string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=true&multiStatements=true",
		i.options.rootUsername(), i.options.rootPassword(), host, i.mappedPort, dbName)
}

func (i *image) startContainer() (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("start container: %w", err)
		}
	}()
	ctx := context.Background()
	dbPort := i.options.defaultPort()
	natPort := nat.Port(dbPort + "/tcp")
	req := testcontainers.ContainerRequest{
		Image:        i.options.useImage(),
		ExposedPorts: []string{dbPort + "/tcp"},
		Cmd:          []string{"--default-authentication-plugin=mysql_native_password"},
		WaitingFor: wait.ForAll(
			wait.ForLog("port: "+dbPort+"  MySQL Community Server - GPL"),
			wait.ForListeningPort(natPort)),
		Env: map[string]string{
			"MYSQL_ROOT_USER":     i.options.rootUsername(),
			"MYSQL_ROOT_PASSWORD": i.options.rootPassword(),
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

func (i *image) shutdown() {
	if i.db != nil {
		_ = i.db.Close()
	}
	if i.container != nil && !i.options.LeaveRunning {
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

const ImageName = "mysql"

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
	return i.options.rootUsername()
}

func (i *image) Password() string {
	return i.options.rootPassword()
}
