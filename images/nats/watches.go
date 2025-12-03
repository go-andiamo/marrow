package nats

import (
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	nc "github.com/nats-io/nats.go"
	"sync"
	"time"
)

// BucketWatch is a before operation that starts a Nats KeyValue watch listener on a specified bucket
//
// the name identifies the listener - for use in marrow.Events, marrow.EventsCount and marrow.EventsClear
//
// if a listener with that name has previously been created, it is cleared
//
//go:noinline
func BucketWatch(name string, bucket any, imgName ...string) marrow.BeforeAfter {
	result := &kvListener{
		capture: capture{
			name:    name,
			when:    marrow.Before,
			imgName: imgName,
			frame:   framing.NewFrame(0),
		},
		listenerName: name,
		bucket:       bucket,
	}
	result.run = result.runListener
	return result
}

type kvListener struct {
	capture
	listenerName string
	bucket       any
	msgs         []any
	mutex        sync.RWMutex
	kw           nc.KeyWatcher
}

var _ marrow.Listener = (*kvListener)(nil)

func (k *kvListener) runListener(ctx marrow.Context, img *image) (err error) {
	if existing := ctx.Listener(k.listenerName); existing == nil {
		var bv any
		if bv, err = marrow.ResolveValue(k.bucket, ctx); err == nil {
			bn := fmt.Sprintf("%v", bv)
			var js nc.JetStreamContext
			if js, err = img.client.JetStream(); err == nil {
				var kvs nc.KeyValue
				if kvs, err = js.KeyValue(bn); err == nil {
					if k.kw, err = kvs.WatchAll(); err == nil {
						go k.receive()
						ctx.RegisterListener(k.listenerName, k)
					}
				}
			}
		}
	} else if _, ok := existing.(*kvListener); !ok {
		err = fmt.Errorf("expected kvListener but got %T", existing)
	} else {
		existing.Clear()
	}
	return err
}

func (k *kvListener) Events() []any {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	cp := make([]any, len(k.msgs))
	copy(cp, k.msgs)
	return cp
}

func (k *kvListener) EventsCount() int {
	k.mutex.RLock()
	defer k.mutex.RUnlock()
	return len(k.msgs)
}

func (k *kvListener) Clear() {
	k.mutex.Lock()
	defer k.mutex.Unlock()
	k.msgs = make([]any, 0)
}

func (k *kvListener) Stop() {
	if k.kw != nil {
		_ = k.kw.Stop()
	}
}

func (k *kvListener) receive() {
	if k.kw == nil {
		return
	}
	for upd := range k.kw.Updates() {
		if upd == nil {
			continue
		}
		k.mutex.Lock()
		k.msgs = append(k.msgs, map[string]any{
			"bucket":    upd.Bucket(),
			"key":       upd.Key(),
			"value":     upd.Value(),
			"revision":  int64(upd.Revision()),
			"created":   upd.Created().Format(time.RFC3339Nano),
			"operation": upd.Operation().String(),
			"op":        int(upd.Operation()),
		})
		k.mutex.Unlock()
	}
}
