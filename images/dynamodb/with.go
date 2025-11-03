package dynamodb

import (
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/with"
	"github.com/testcontainers/testcontainers-go"
)

type Image interface {
	with.With
	Start() error
	MappedPort() string
	Database() *sql.DB
	Client() *dynamodb.Client
	Container() testcontainers.Container
}

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
		return fmt.Errorf("with dynamodb image init error: %w", err)
	}
	init.AddDb(i.name, i.db, common.PositionalDbArgs)
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
