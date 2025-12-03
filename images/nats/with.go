package nats

import (
	"fmt"
	"github.com/go-andiamo/marrow/with"
	nc "github.com/nats-io/nats.go"
	"github.com/testcontainers/testcontainers-go"
)

type Image interface {
	with.With
	Start() error
	MappedPort() string
	Container() testcontainers.Container
	Client() *nc.Conn
}

// With creates a new Nats support image for use in marrow.Suite .Init()
func With(options Options) Image {
	return &image{options: options}
}

var _ with.With = (*image)(nil)
var _ with.Image = (*image)(nil)
var _ Image = (*image)(nil)

func (i *image) Init(init with.SuiteInit) error {
	if err := i.Start(); err != nil {
		return fmt.Errorf("with nats image init error: %w", err)
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
