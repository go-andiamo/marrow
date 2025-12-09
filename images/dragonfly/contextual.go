package dragonfly

import (
	"encoding/json"
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"reflect"
	"time"
)

// QueueLen can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the number of messages on the named queue
//
// the named queue does not have to have been listened on - for listened queues (i.e. specified in Options.Consumers) use
// ReceivedQueueMessages
//
//go:noinline
func QueueLen(queueName string, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("QueueLen(%q)", queueName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (any, error) {
			return img.Client().QueueLength(queueName)
		},
		frame: framing.NewFrame(0),
	}
}

// SendMessage can be used as a before/after on marrow.Method .Capture
// and sends a message to a dragonfly queue
//
//go:noinline
func SendMessage(when marrow.When, queueName string, message any, imgName ...string) marrow.BeforeAfter {
	return &capture{
		name:    "SendMessage",
		when:    when,
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			var actualMsg any
			if actualMsg, err = marrow.ResolveValue(message, ctx); err == nil {
				err = img.Client().Send(queueName, actualMsg)
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// PublishMessage can be used as a before/after on marrow.Method .Capture
// and publishes a message to a dragonfly topic
//
//go:noinline
func PublishMessage(when marrow.When, topicName string, message any, imgName ...string) marrow.BeforeAfter {
	return &capture{
		name:    "PublishMessage",
		when:    when,
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			var actualMsg any
			if actualMsg, err = marrow.ResolveValue(message, ctx); err == nil {
				err = img.Client().Publish(topicName, actualMsg)
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// ReceivedQueueMessages can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the number of messages received on the named queue
//
// the named queue must have been listened on - i.e. specified in Options.Consumers
//
//go:noinline
func ReceivedQueueMessages(queueName string, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("ReceivedQueueMessages(%q)", queueName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (any, error) {
			if l, ok := img.queueListeners[queueName]; ok {
				return l.received(), nil
			} else {
				return nil, fmt.Errorf("queue %q not listened on", queueName)
			}
		},
		frame: framing.NewFrame(0),
	}
}

// ReceivedQueueMessage can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to received message, by index, on the named queue
//
// the index can be negative - which means offset from last, i.e. -1 is last
//
// the named queue must have been listened on - i.e. specified in Options.Consumers
//
//go:noinline
func ReceivedQueueMessage(queueName string, index int, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("ReceivedQueueMessage(%q, %d)", queueName, index),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (any, error) {
			if l, ok := img.queueListeners[queueName]; ok {
				return l.receivedMessage(index)
			} else {
				return nil, fmt.Errorf("queue %q not listened on", queueName)
			}
		},
		frame: framing.NewFrame(0),
	}
}

// ReceivedTopicMessages can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the number of messages received on the named topic
//
// the named topic must have been listened on - i.e. specified in Options.Subscribers
//
//go:noinline
func ReceivedTopicMessages(topicName string, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("ReceivedTopicMessages(%q)", topicName),
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

// ReceivedTopicMessage can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to received message, by index, on the named topic
//
// the index can be negative - which means offset from last, i.e. -1 is last
//
// the named topic must have been listened on - i.e. specified in Options.Subscribers
//
//go:noinline
func ReceivedTopicMessage(topicName string, index int, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("ReceivedTopicMessage(%q, %d)", topicName, index),
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

func jsonMarshal(v any) (s string) {
	if js, err := json.Marshal(v); err == nil {
		s = string(js)
	}
	return s
}

// SetKey can be used as a before/after on marrow.Method .Capture
// and sets a named key
//
// note: the value can be a resolvable
//
//go:noinline
func SetKey(when marrow.When, name string, value any, expiry time.Duration, imgName ...string) marrow.BeforeAfter {
	return &capture{
		name:    fmt.Sprintf("SetKey(%q)", name),
		when:    when,
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			var av any
			if av, err = marrow.ResolveValue(value, ctx); err == nil {
				sv := ""
				switch avt := av.(type) {
				case string:
					sv = avt
				case []byte:
					sv = string(avt)
				case []any:
					sv = jsonMarshal(av)
				case map[string]any:
					sv = jsonMarshal(av)
				default:
					if av != nil {
						to := reflect.ValueOf(av)
						if to.Kind() == reflect.Slice || to.Kind() == reflect.Map || to.Kind() == reflect.Struct {
							sv = jsonMarshal(av)
						} else {
							sv = fmt.Sprintf("%v", av)
						}
					}
				}
				err = img.Client().Set(name, sv, expiry)
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// DeleteKey can be used as a before/after on marrow.Method .Capture
// and deletes a named key
//
//go:noinline
func DeleteKey(when marrow.When, name string, imgName ...string) marrow.BeforeAfter {
	return &capture{
		name:    fmt.Sprintf("DeleteKey(%q)", name),
		when:    when,
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			_, err = img.Client().Delete(name)
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// Key can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the value of the named key
//
//go:noinline
func Key(name string, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("Key(%q)", name),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (any, error) {
			return img.Client().Get(name)
		},
		frame: framing.NewFrame(0),
	}
}

// KeyExists can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to a bool of whether the named key exists
//
//go:noinline
func KeyExists(name string, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("Key(%q)", name),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (any, error) {
			return img.Client().Exists(name)
		},
		frame: framing.NewFrame(0),
	}
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
