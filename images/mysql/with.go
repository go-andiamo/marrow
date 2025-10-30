package mysql

import (
	"database/sql"
	"fmt"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/with"
	"github.com/testcontainers/testcontainers-go"
)

type DbImage interface {
	with.With
	Start() error
	MappedPort() string
	Database() *sql.DB
	Container() testcontainers.Container
}

func WithDbImage(options Options) DbImage {
	return &image{options: options}
}

var _ with.With = (*image)(nil)
var _ DbImage = (*image)(nil)

func (i *image) Init(init with.SuiteInit) error {
	if err := i.Start(); err != nil {
		return fmt.Errorf("with mysql image init error: %w", err)
	}
	init.SetDb(i.db)
	init.SetDbArgMarkers(common.PositionalDbArgs)
	init.AddSupportingImage(with.ImageInfo{
		Name:       "mysql",
		Host:       "localhost",
		Port:       i.options.defaultPort(),
		MappedPort: i.mappedPort,
		IsDocker:   true,
		Username:   i.options.username(),
		Password:   i.options.password(),
	})
	return nil
}

func (i *image) Stage() with.Stage {
	return with.Supporting
}

func (i *image) Shutdown() func() {
	return func() {
		i.shutdown()
	}
}
