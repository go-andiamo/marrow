package nats

import (
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	nc "github.com/nats-io/nats.go"
	"sync"
)

// Subscribe is a before operation that starts a Nats subscription on a specified subject
//
// the name identifies the listener - for use in marrow.Events, marrow.EventsCount and marrow.EventsClear
//
// if a listener with that name has previously been created, it is cleared
//
//go:noinline
func Subscribe(name string, subject any, imgName ...string) marrow.BeforeAfter {
	result := &subjectListener{
		capture: capture{
			name:    name,
			when:    marrow.Before,
			imgName: imgName,
			frame:   framing.NewFrame(0),
		},
		listenerName: name,
		subject:      subject,
	}
	result.run = result.runListener
	return result
}

type subjectListener struct {
	capture
	listenerName string
	subject      any
	msgs         []any
	mutex        sync.RWMutex
	sub          *nc.Subscription
}

var _ marrow.Listener = (*subjectListener)(nil)

func (l *subjectListener) runListener(ctx marrow.Context, img *image) (err error) {
	if existing := ctx.Listener(l.listenerName); existing == nil {
		var sv any
		if sv, err = marrow.ResolveValue(l.subject, ctx); err == nil {
			subject := fmt.Sprintf("%v", sv)
			if l.sub, err = img.client.Subscribe(subject, l.handler); err == nil {
				ctx.RegisterListener(l.listenerName, l)
			}
		}
	} else if _, ok := existing.(*subjectListener); !ok {
		err = fmt.Errorf("expected subjectListener but got %T", existing)
	} else {
		existing.Clear()
	}
	return err
}

func (l *subjectListener) handler(msg *nc.Msg) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.msgs = append(l.msgs, msg.Data)
}

func (l *subjectListener) Events() []any {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	cp := make([]any, len(l.msgs))
	copy(cp, l.msgs)
	return cp
}

func (l *subjectListener) EventsCount() int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return len(l.msgs)
}

func (l *subjectListener) Clear() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.msgs = make([]any, 0)
}

func (l *subjectListener) Stop() {
	if l.sub != nil {
		_ = l.sub.Unsubscribe()
	}
}
