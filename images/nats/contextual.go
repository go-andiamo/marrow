package nats

import (
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	nc "github.com/nats-io/nats.go"
)

// Publish can be used as a before/after on marrow.Method .Capture
// and publishes a message to a subject
//
// note: both the subject and msg value can be a resolvable
//
//go:noinline
func Publish(when marrow.When, subject any, msg any, imgName ...string) marrow.BeforeAfter {
	return &capture{
		name:    fmt.Sprintf("Publish(%v)", subject),
		when:    when,
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			var sv any
			if sv, err = marrow.ResolveValue(subject, ctx); err == nil {
				subj := fmt.Sprintf("%v", sv)
				var data []byte
				if data, err = marrow.ResolveData(msg, ctx); err == nil {
					err = img.client.Publish(subj, data)
				}
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// Key can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the value of the named key
//
//go:noinline
func Key(bucket any, name any, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("Key(%v, %v)", bucket, name),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (av any, err error) {
			var bv any
			if bv, err = marrow.ResolveValue(bucket, ctx); err == nil {
				bn := fmt.Sprintf("%v", bv)
				var kv any
				if kv, err = marrow.ResolveValue(name, ctx); err == nil {
					kn := fmt.Sprintf("%v", kv)
					var js nc.JetStreamContext
					if js, err = img.client.JetStream(); err == nil {
						var kvs nc.KeyValue
						if kvs, err = js.KeyValue(bn); err == nil {
							var kve nc.KeyValueEntry
							if kve, err = kvs.Get(kn); err == nil {
								av = kve.Value()
							}
						}
					}
				}
			}
			return av, err
		},
		frame: framing.NewFrame(0),
	}
}

// KeyExists can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to whether the named key exists
//
//go:noinline
func KeyExists(bucket any, name any, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("Key(%v, %v)", bucket, name),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (av any, err error) {
			var bv any
			if bv, err = marrow.ResolveValue(bucket, ctx); err == nil {
				bn := fmt.Sprintf("%v", bv)
				var kv any
				if kv, err = marrow.ResolveValue(name, ctx); err == nil {
					kn := fmt.Sprintf("%v", kv)
					var js nc.JetStreamContext
					if js, err = img.client.JetStream(); err == nil {
						var kvs nc.KeyValue
						if kvs, err = js.KeyValue(bn); err == nil {
							if _, err = kvs.Get(kn); err == nil {
								av = true
							} else if err == nc.ErrKeyNotFound {
								err = nil
								av = false
							}
						}
					}
				}
			}
			return av, err
		},
		frame: framing.NewFrame(0),
	}
}

// PutKey can be used as a before/after on marrow.Method .Capture
// and puts a named key in the specified bucket
//
// note: the value can be a resolvable
//
//go:noinline
func PutKey(when marrow.When, bucket any, name any, value any, imgName ...string) marrow.BeforeAfter {
	return &capture{
		name:    fmt.Sprintf("PutKey(%v, %v)", bucket, name),
		when:    when,
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			var bv any
			if bv, err = marrow.ResolveValue(bucket, ctx); err == nil {
				bn := fmt.Sprintf("%v", bv)
				var kv any
				if kv, err = marrow.ResolveValue(name, ctx); err == nil {
					kn := fmt.Sprintf("%v", kv)
					var data []byte
					if data, err = marrow.ResolveData(value, ctx); err == nil {
						var js nc.JetStreamContext
						if js, err = img.client.JetStream(); err == nil {
							var kvs nc.KeyValue
							if kvs, err = js.KeyValue(bn); err == nil {
								_, err = kvs.Put(kn, data)
							}
						}
					}
				}
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// DeleteKey can be used as a before/after on marrow.Method .Capture
// and deletes a named key in the specified bucket
//
//go:noinline
func DeleteKey(when marrow.When, bucket any, name any, imgName ...string) marrow.BeforeAfter {
	return &capture{
		name:    fmt.Sprintf("DeleteKey(%v, %v)", bucket, name),
		when:    when,
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			var bv any
			if bv, err = marrow.ResolveValue(bucket, ctx); err == nil {
				bn := fmt.Sprintf("%v", bv)
				var kv any
				if kv, err = marrow.ResolveValue(name, ctx); err == nil {
					kn := fmt.Sprintf("%v", kv)
					var js nc.JetStreamContext
					if js, err = img.client.JetStream(); err == nil {
						var kvs nc.KeyValue
						if kvs, err = js.KeyValue(bn); err == nil {
							err = kvs.Delete(kn)
						}
					}
				}
			}
			return err
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
	return imageName + "." + r.name
}

func imageFromContext(ctx marrow.Context, name []string) (*image, error) {
	n := imageName
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
