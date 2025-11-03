package dragonfly

import (
	"fmt"
	"github.com/go-andiamo/marrow/with"
	"github.com/testcontainers/testcontainers-go"
)

type Image interface {
	with.With
	Start() error
	MappedPort() string
	Container() testcontainers.Container
}

func With(options Options) Image {
	return &image{options: options}
}

var _ with.With = (*image)(nil)
var _ with.Image = (*image)(nil)
var _ Image = (*image)(nil)

func (i *image) Init(init with.SuiteInit) error {
	if err := i.Start(); err != nil {
		return fmt.Errorf("with dragonfly image init error: %w", err)
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
