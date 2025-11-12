package localstack

import (
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
)

type capture[T any] struct {
	name     string
	when     marrow.When
	defImage string
	imgName  []string
	run      func(ctx marrow.Context, img T) error
	frame    *framing.Frame
}

var _ marrow.BeforeAfter = (*capture[any])(nil)

func (c *capture[T]) When() marrow.When {
	return c.when
}

func (c *capture[T]) Run(ctx marrow.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running operation %s: %w", c.name, err)
		}
	}()
	var img T
	if img, err = imageFromContext[T](ctx, c.defImage, c.imgName); err == nil {
		err = c.run(ctx, img)
	}
	return err
}

func (c *capture[T]) Frame() *framing.Frame {
	return c.frame
}

type resolvable[T any] struct {
	name     string
	defImage string
	imgName  []string
	run      func(ctx marrow.Context, img T) (any, error)
	frame    *framing.Frame
}

var _ marrow.Resolvable = (*resolvable[any])(nil)
var _ fmt.Stringer = (*resolvable[any])(nil)

func (r *resolvable[T]) ResolveValue(ctx marrow.Context) (av any, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error resolving value %s: %w", r.name, err)
		}
	}()
	var img T
	if img, err = imageFromContext[T](ctx, r.defImage, r.imgName); err == nil {
		av, err = r.run(ctx, img)
	}
	return av, err
}

func (r *resolvable[T]) String() string {
	return r.defImage + "." + r.name
}

func imageFromContext[T any](ctx marrow.Context, defImage string, name []string) (T, error) {
	n := defImage
	if len(name) > 0 && name[0] != "" {
		n = name[0]
	}
	if i := ctx.GetImage(n); i != nil {
		if img, ok := i.(T); ok {
			return img, nil
		}
	}
	var z T
	return z, fmt.Errorf("image not found: %s", name)
}
