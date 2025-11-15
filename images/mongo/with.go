package mongo

import (
	"fmt"
	"github.com/go-andiamo/marrow/with"
	"github.com/testcontainers/testcontainers-go"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Image interface {
	with.With
	Start() error
	MappedPort() string
	Container() testcontainers.Container
	Client() *mongo.Client
}

// With creates a new MongoDB support image for use in marrow.Suite .Init()
func With(options Options) Image {
	return newImage(options)
}

var _ with.With = (*image)(nil)
var _ with.Image = (*image)(nil)
var _ Image = (*image)(nil)

func (i *image) Init(init with.SuiteInit) error {
	if err := i.Start(); err != nil {
		return fmt.Errorf("with mongo image init error: %w", err)
	}
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
