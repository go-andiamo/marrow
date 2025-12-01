package artemis

import (
	"encoding/json"
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-stomp/stomp/v3"
	"math"
	"sync"
)

func (i *image) setupListeners() (err error) {
	i.topicListeners = make(map[string]*listener, len(i.options.Subscribers))
	i.queueListeners = make(map[string]*listener, len(i.options.Consumers))
	for k, v := range i.options.Subscribers {
		ln := v.MaxMessages
		if ln < 0 {
			ln = 0
		}
		l := &listener{
			max:         ln,
			json:        v.JsonMessages,
			unmarshaler: v.Unmarshaler,
			msgs:        make([]any, 0, ln),
		}
		i.topicListeners[k] = l
		if l.close, err = i.client.Subscribe(k, l.receive); err != nil {
			return err
		}
	}
	for k, v := range i.options.Consumers {
		ln := v.MaxMessages
		if ln < 0 {
			ln = 0
		}
		l := &listener{
			max:         ln,
			json:        v.JsonMessages,
			unmarshaler: v.Unmarshaler,
			msgs:        make([]any, 0, ln),
		}
		i.queueListeners[k] = l
		if l.close, err = i.client.Consume(k, l.receive); err != nil {
			return err
		}
	}
	return nil
}

// TopicListener is a before operation that starts a topic listener
//
// the name identifies the listener - for use in marrow.Events and marrow.EventsClear
//
// if a listener with that name has previously been created, it is cleared
//
//go:noinline
func TopicListener(name string, topic string, options Receiver, imgName ...string) marrow.BeforeAfter {
	name, topic = nameAndDest(name, topic)
	result := &topicListener{
		capture: capture{
			name:    fmt.Sprintf("TopicListener(%q)", topic),
			when:    marrow.Before,
			imgName: imgName,
			frame:   framing.NewFrame(0),
		},
		listenerName: name,
		topic:        topic,
		options:      options,
	}
	result.run = result.runListener
	return result
}

// QueueListener is a before operation that starts a queue listener
//
// the name identifies the listener - for use in marrow.Events and marrow.EventsClear
//
// if a listener with that name has previously been created, it is cleared
//
//go:noinline
func QueueListener(name string, queue string, options Receiver, imgName ...string) marrow.BeforeAfter {
	name, queue = nameAndDest(name, queue)
	result := &queueListener{
		capture: capture{
			name:    fmt.Sprintf("QueueListener(%q)", queue),
			when:    marrow.Before,
			imgName: imgName,
			frame:   framing.NewFrame(0),
		},
		listenerName: name,
		queue:        queue,
		options:      options,
	}
	result.run = result.runListener
	return result
}

func nameAndDest(name string, dst string) (string, string) {
	if name == "" && dst != "" {
		return dst, dst
	}
	if name != "" && dst == "" {
		return name, name
	}
	return name, dst
}

type topicListener struct {
	capture
	listenerName string
	topic        string
	options      Receiver
}

func (t *topicListener) runListener(ctx marrow.Context, img *image) (err error) {
	if existing := ctx.Listener(t.listenerName); existing == nil {
		ln := t.options.MaxMessages
		if ln <= 0 {
			ln = math.MaxInt
		}
		l := &listener{
			max:         ln,
			json:        t.options.JsonMessages,
			unmarshaler: t.options.Unmarshaler,
		}
		if l.close, err = img.Client().Subscribe(t.topic, l.receive); err == nil {
			ctx.RegisterListener(t.listenerName, l)
		}
	} else if _, ok := existing.(*listener); !ok {
		err = fmt.Errorf("expected topicListener but got %T", existing)
	} else {
		existing.Clear()
	}
	return err
}

type queueListener struct {
	capture
	listenerName string
	queue        string
	options      Receiver
}

func (q *queueListener) runListener(ctx marrow.Context, img *image) (err error) {
	if existing := ctx.Listener(q.listenerName); existing == nil {
		ln := q.options.MaxMessages
		if ln <= 0 {
			ln = math.MaxInt
		}
		l := &listener{
			max:         ln,
			json:        q.options.JsonMessages,
			unmarshaler: q.options.Unmarshaler,
		}
		if l.close, err = img.Client().Consume(q.queue, l.receive); err == nil {
			ctx.RegisterListener(q.listenerName, l)
		}
	} else if _, ok := existing.(*listener); !ok {
		err = fmt.Errorf("expected queueListener but got %T", existing)
	} else {
		existing.Clear()
	}
	return err
}

type listener struct {
	count       int64
	msgs        []any
	max         int
	json        bool
	unmarshaler func(msg *stomp.Message) any
	close       func()
	mutex       sync.RWMutex
}

var _ marrow.Listener = (*listener)(nil)

func (l *listener) Events() []any {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	cp := make([]any, len(l.msgs))
	copy(cp, l.msgs)
	return cp
}

func (l *listener) EventsCount() int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return len(l.msgs)
}

func (l *listener) Clear() {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.msgs = make([]any, 0)
}

func (l *listener) Stop() {
	l.close()
}

func (l *listener) receive(msg *stomp.Message) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.count == math.MaxInt64 {
		l.count = 1
	} else {
		l.count++
	}
	if l.max > 0 {
		var v any = msg
		if l.unmarshaler != nil {
			v = l.unmarshaler(msg)
		} else if l.json {
			var jv any
			if err := json.Unmarshal(msg.Body, &jv); err == nil {
				v = jv
			}
		}
		if len(l.msgs) < l.max {
			l.msgs = append(l.msgs, v)
		} else {
			// drop oldest and append newest...
			l.msgs[0] = nil
			copy(l.msgs, l.msgs[1:])
			l.msgs[len(l.msgs)-1] = v
		}
	}
}

func (l *listener) stop() {
	if l.close != nil {
		l.close()
	}
	l.close = nil
}

func (l *listener) received() int64 {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.count
}

func (l *listener) receivedMessage(index int) (any, error) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	idx := index
	if idx < 0 {
		idx = len(l.msgs) + idx
	}
	if idx >= 0 && idx < len(l.msgs) {
		return l.msgs[idx], nil
	} else {
		return nil, fmt.Errorf("message index out of range %d", index)
	}
}
