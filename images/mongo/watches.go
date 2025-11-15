package mongo

import (
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"math"
	"sync"
)

func newWatch(changeStream *mongo.ChangeStream, max int) *watch {
	ln := max
	if ln < 0 {
		ln = 0
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := &watch{
		max:          max,
		changes:      make([]map[string]any, 0, ln),
		changeStream: changeStream,
		ctx:          ctx,
		cancel:       cancel,
	}
	go result.listen()
	return result
}

type watch struct {
	count        int64
	max          int
	changes      []map[string]any
	changeStream *mongo.ChangeStream
	ctx          context.Context
	cancel       context.CancelFunc
	mutex        sync.RWMutex
}

func (w *watch) listen() {
	defer func() {
		_ = w.changeStream.Close(context.Background())
	}()
	for w.changeStream.Next(w.ctx) {
		chg := map[string]any{}
		_ = w.changeStream.Decode(&chg)
		normalizeJson(chg)
		w.mutex.Lock()
		if w.count == math.MaxInt64 {
			w.count = 1
		} else {
			w.count++
		}
		if w.max > 0 {
			if len(w.changes) < w.max {
				w.changes = append(w.changes, chg)
			} else {
				// drop oldest and append newest (nil for clarity - copy + overwrite already releases)
				w.changes[0] = nil
				copy(w.changes, w.changes[1:])
				w.changes[len(w.changes)-1] = chg
			}
		}
		w.mutex.Unlock()
	}
}

func normalizeJson(in map[string]any) {
	for k, v := range in {
		switch vt := v.(type) {
		case bson.D:
			m := make(map[string]any)
			if err := json.Unmarshal([]byte(vt.String()), &m); err == nil {
				in[k] = m
			}
		}
	}
}

func (w *watch) stop() {
	w.cancel()
	_ = w.changeStream.Close(context.Background())
}

func (w *watch) countChanges() int64 {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	return w.count
}

func (w *watch) changedItems(op string) any {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	changes := append([]map[string]any{}, w.changes...)
	if op != "" && op != "*" {
		changes = make([]map[string]any, 0, len(w.changes))
		for _, chg := range w.changes {
			rawOt := chg["operationType"]
			if ot, ok := rawOt.(string); ok && ot == op {
				changes = append(changes, chg)
			}
		}
	}
	return changes
}
