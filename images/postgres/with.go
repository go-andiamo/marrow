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

// With creates a new Postgres support image for use in marrow.Suite .Init()
func With(name string, options Options) Image {
	return &image{
		name:    name,
		options: options,
	}
}

var _ with.With = (*image)(nil)
var _ with.Image = (*image)(nil)
var _ Image = (*image)(nil)

func (i *image) Init(init with.SuiteInit) error {
	if err := i.Start(); err != nil {
		return fmt.Errorf("with mysql image init error: %w", err)
	}
	init.AddDb(i.name, i.db, common.NumberedDbArgs)
	init.AddSupportingImage(i)
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
