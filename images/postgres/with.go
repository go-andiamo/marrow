package postgres

import (
	"database/sql"
	"fmt"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/with"
	"github.com/testcontainers/testcontainers-go"
)

type Image interface {
	with.With
	Start() error
	MappedPort() string
	Database() *sql.DB
	Container() testcontainers.Container
}

func With(options Options) Image {
	return &image{options: options}
}

var _ with.With = (*image)(nil)
var _ Image = (*image)(nil)

func (i *image) Init(init with.SuiteInit) error {
	if err := i.Start(); err != nil {
		return fmt.Errorf("with mysql image init error: %w", err)
	}
	init.SetDb(i.db)
	init.SetDbArgMarkers(common.NumberedDbArgs)
	init.AddSupportingImage(with.ImageInfo{
		Name:       "postgres",
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
