package kafka

import (
	"encoding/json"
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"reflect"
)

// Publish can be used as a before/after on marrow.Method .Capture
// and publishes a message to a kafka topic
//
//go:noinline
func Publish(when marrow.When, topicName string, key any, value any, imgName ...string) marrow.BeforeAfter {
	return &capture{
		name:    "Publish",
		when:    when,
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			var avk any
			if avk, err = marrow.ResolveValue(key, ctx); err == nil {
				ks := stringify(avk)
				var avv any
				if avv, err = marrow.ResolveValue(value, ctx); err == nil {
					vs := stringify(avv)
					err = img.Client().Publish(topicName, ks, vs)
				}
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// ReceivedMessages can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the number of messages received on the named topic
//
// the named topic must have been listened on - i.e. specified in Options.Subscribers
//
//go:noinline
func ReceivedMessages(topicName string, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("ReceivedMessages(%q)", topicName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (any, error) {
			if l, ok := img.topicListeners[topicName]; ok {
				return l.received(), nil
			} else {
				return nil, fmt.Errorf("topic %q not listened on", topicName)
			}
		},
		frame: framing.NewFrame(0),
	}
}

// ReceivedMessage can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to received message, by index, on the named topic
//
// the index can be negative - which means offset from last, i.e. -1 is last
//
// the named topic must have been listened on - i.e. specified in Options.Subscribers
//
//go:noinline
func ReceivedMessage(topicName string, index int, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("ReceivedMessage(%q, %d)", topicName, index),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (any, error) {
			if l, ok := img.topicListeners[topicName]; ok {
				return l.receivedMessage(index)
			} else {
				return nil, fmt.Errorf("topic %q not listened on", topicName)
			}
		},
		frame: framing.NewFrame(0),
	}
}

func stringify(v any) (s string) {
	switch vt := v.(type) {
	case string:
		s = vt
	case []byte:
		s = string(vt)
	case []any:
		s = jsonMarshal(v)
	case map[string]any:
		s = jsonMarshal(v)
	default:
		if v != nil {
			to := reflect.ValueOf(v)
			if to.Kind() == reflect.Slice || to.Kind() == reflect.Map || to.Kind() == reflect.Struct {
				s = jsonMarshal(v)
			} else {
				s = fmt.Sprintf("%v", v)
			}
		}
	}
	return s
}

func jsonMarshal(v any) (s string) {
	if js, err := json.Marshal(v); err == nil {
		s = string(js)
	} else {
		s = fmt.Sprintf("%v", v)
	}
	return s
}

type capture struct {
	name    string
	when    marrow.When
	imgName []string
	run     func(ctx marrow.Context, img *image) error
	frame   *framing.Frame
}

var _ marrow.BeforeAfter = (*capture)(nil)

func (c *capture) When() marrow.When {
	return c.when
}

func (c *capture) Run(ctx marrow.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running operation %s: %w", c.name, err)
		}
	}()
	var img *image
	if img, err = imageFromContext(ctx, c.imgName); err == nil {
		err = c.run(ctx, img)
	}
	return err
}

func (c *capture) Frame() *framing.Frame {
	return c.frame
}

type resolvable struct {
	name    string
	imgName []string
	run     func(ctx marrow.Context, img *image) (any, error)
	frame   *framing.Frame
}

var _ marrow.Resolvable = (*resolvable)(nil)
var _ fmt.Stringer = (*resolvable)(nil)

func (r *resolvable) ResolveValue(ctx marrow.Context) (av any, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error resolving value %s: %w", r.name, err)
		}
	}()
	var img *image
	if img, err = imageFromContext(ctx, r.imgName); err == nil {
		av, err = r.run(ctx, img)
	}
	return av, err
}

func (r *resolvable) String() string {
	return ImageName + "." + r.name
}

func imageFromContext(ctx marrow.Context, name []string) (*image, error) {
	n := ImageName
	if len(name) > 0 && name[0] != "" {
		n = name[0]
	}
	if i := ctx.GetImage(n); i != nil {
		if img, ok := i.(*image); ok {
			return img, nil
		}
	}
	return nil, fmt.Errorf("image not found: %s", name)
}
