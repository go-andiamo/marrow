package kafka

import (
	"encoding/json"
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"math"
	"sync"
)

func (i *image) setupListeners() error {
	i.topicListeners = make(map[string]*listener, len(i.options.Subscribers))
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
		l.close = i.client.Subscribe(k, l.receive)
	}
	return nil
}

// TopicListener is a before operation that starts a Kafka listener
//
// the name identifies the listener - for use in marrow.Events and marrow.EventsClear
//
// if a listener with that name has previously been created, it is cleared
//
//go:noinline
func TopicListener(name string, topic string, options Subscriber, imgName ...string) marrow.BeforeAfter {
	name, topic = nameAndDest(name, topic)
	result := &kafkaListener{
		capture: capture{
			name:    fmt.Sprintf("Listener(%q)", topic),
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

func nameAndDest(name string, dst string) (string, string) {
	if name == "" && dst != "" {
		return dst, dst
	}
	if name != "" && dst == "" {
		return name, name
	}
	return name, dst
}

type kafkaListener struct {
	capture
	listenerName string
	topic        string
	options      Subscriber
}

func (k *kafkaListener) runListener(ctx marrow.Context, img *image) (err error) {
	if existing := ctx.Listener(k.listenerName); existing == nil {
		ln := k.options.MaxMessages
		if ln <= 0 {
			ln = math.MaxInt
		}
		l := &listener{
			max:         ln,
			mark:        k.options.Mark,
			json:        k.options.JsonMessages,
			unmarshaler: k.options.Unmarshaler,
		}
		l.close = img.Client().Subscribe(k.topic, l.receive)
		ctx.RegisterListener(k.listenerName, l)
	} else if _, ok := existing.(*listener); !ok {
		err = fmt.Errorf("expected kafkaListener but got %T", existing)
	} else {
		existing.Clear()
	}
	return err
}

type listener struct {
	mark        string
	count       int64
	msgs        []any
	max         int
	json        bool
	unmarshaler func(msg Message) any
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

func (l *listener) receive(msg Message) (mark string) {
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
			hdrs := make(map[string]any, len(msg.Headers))
			for _, hdr := range msg.Headers {
				hdrs[string(hdr.Key)] = string(hdr.Value)
			}
			mv := map[string]any{
				"key":            string(msg.Key),
				"value":          string(msg.Value),
				"timestamp":      msg.Timestamp,
				"blockTimestamp": msg.BlockTimestamp,
				"topic":          msg.Topic,
				"partition":      msg.Partition,
				"offset":         msg.Offset,
				"headers":        hdrs,
			}
			var jv any
			if err := json.Unmarshal([]byte(msg.Value), &jv); err == nil {
				mv["value"] = jv
			}
			v = mv
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
	return l.mark
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
