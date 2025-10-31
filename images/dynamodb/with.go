package dynamodb

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/go-andiamo/marrow/with"
	"github.com/testcontainers/testcontainers-go"
)

type Image interface {
	with.With
	Start() error
	MappedPort() string
	Container() testcontainers.Container
	Client() *dynamodb.Client
}

func With(options Options) Image {
	return &image{options: options}
}

var _ with.With = (*image)(nil)
var _ Image = (*image)(nil)

func (i *image) Init(init with.SuiteInit) error {
	if err := i.Start(); err != nil {
		return fmt.Errorf("with dynamodb image init error: %w", err)
	}
	init.AddSupportingImage(with.ImageInfo{
		Name:       "dynamodb",
		Host:       "localhost",
		Port:       i.options.defaultPort(),
		MappedPort: i.mappedPort,
		IsDocker:   true,
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
