package kafka

import (
	"encoding/json"
	"fmt"
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
		i.client.Subscribe(k, l.receive)
	}
	return nil
}

type listener struct {
	mark        string
	count       int64
	msgs        []any
	max         int
	json        bool
	unmarshaler func(msg Message) any
	mutex       sync.RWMutex
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
