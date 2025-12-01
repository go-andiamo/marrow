package marrow

import (
	"bufio"
	gctx "context"
	"fmt"
	"github.com/go-andiamo/marrow/framing"
	"net/http"
	"strings"
	"sync"
)

// Events is a resolvable value that resolves to the events on the named listener
//
// if the named listener has not been started/registered - an error is returned
type Events string

func (v Events) ResolveValue(ctx Context) (av any, err error) {
	if l := ctx.Listener(string(v)); l != nil {
		av = l.Events()
	} else {
		err = fmt.Errorf("no event listener found for %q", v)
	}
	return av, err
}

func (v Events) String() string {
	return "Events(" + string(v) + ")"
}

// EventsCount is a resolvable value that resolves to the count of events on the named listener
//
// if the named listener has not been started/registered - an error is returned
type EventsCount string

func (v EventsCount) ResolveValue(ctx Context) (av any, err error) {
	if l := ctx.Listener(string(v)); l != nil {
		av = l.EventsCount()
	} else {
		err = fmt.Errorf("no event listener found for %q", v)
	}
	return av, err
}

func (v EventsCount) String() string {
	return "EventsCount(" + string(v) + ")"
}

type eventsClear struct {
	name  string
	frame *framing.Frame
}

// EventsClear clears the events on the named listener
//
//go:noinline
func EventsClear(name string) Capture {
	return &eventsClear{
		name:  name,
		frame: framing.NewFrame(0),
	}
}

func (e *eventsClear) Name() string {
	return fmt.Sprintf("EventsClear(%s)", e.name)
}

func (e *eventsClear) Run(ctx Context) error {
	if l := ctx.Listener(e.name); l != nil {
		l.Clear()
		return nil
	}
	return fmt.Errorf("no event listener found for %q", e.name)
}

func (e *eventsClear) Frame() *framing.Frame {
	return e.frame
}

type Listener interface {
	Events() []any
	EventsCount() int
	Clear()
	Stop()
}

// SSEListener starts an SSE listener as a before operation
//
// the name identifies the listener - for use in Events and EventsClear
//
// if a listener with that name has previously been created, it is cleared
//
//go:noinline
func SSEListener(name string, url any) BeforeAfter {
	return &sseListener{
		name:  name,
		url:   url,
		frame: framing.NewFrame(0),
	}
}

func (s *sseListener) When() When {
	return Before
}

func (s *sseListener) Run(ctx Context) (err error) {
	if existing := ctx.Listener(s.name); existing == nil {
		if err = s.start(ctx); err == nil {
			ctx.RegisterListener(s.name, s)
		}
	} else if _, ok := existing.(*sseListener); !ok {
		err = fmt.Errorf("expected sseListener but got %T", existing)
	} else {
		existing.Clear()
	}
	return err
}

func (s *sseListener) Frame() *framing.Frame {
	return s.frame
}

func (s *sseListener) String() string {
	return fmt.Sprintf("SSEListener(%s, %v)", s.name, s.url)
}

type sseListener struct {
	name    string
	url     any
	frame   *framing.Frame
	started bool
	cancel  gctx.CancelFunc
	stop    chan struct{}
	mutex   sync.RWMutex
	events  []any
}

var _ Listener = (*sseListener)(nil)

func (s *sseListener) Events() []any {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	cp := make([]any, len(s.events))
	copy(cp, s.events)
	return cp
}

func (s *sseListener) EventsCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.events)
}

func (s *sseListener) Clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.events = nil
}

func (s *sseListener) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.started {
		if s.cancel != nil {
			s.cancel()
		}
		if s.stop != nil {
			select {
			case <-s.stop:
				// already closed, do nothing
			default:
				close(s.stop)
			}
		}
		s.started = false
	}
}

func (s *sseListener) start(ctx Context) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.started {
		s.stop = nil
		s.cancel = nil
		var ap any
		if ap, err = ResolveValue(s.url, ctx); err == nil {
			actualUrl := fmt.Sprintf("http://%s%v", ctx.Host(), ap)
			var reqCtx gctx.Context
			reqCtx, s.cancel = gctx.WithCancel(ctx.Ctx())
			var req *http.Request
			if req, err = http.NewRequestWithContext(reqCtx, http.MethodGet, actualUrl, nil); err == nil {
				req.Header.Set("Accept", "text/event-stream")
				s.stop = make(chan struct{})
				s.started = true
				go s.listen(ctx, req)
			}
		}
	}
	return err
}

func (s *sseListener) listen(ctx Context, req *http.Request) {
	resp, err := ctx.DoRequest(req)
	if err != nil {
		ctx.Log(fmt.Sprintf("SSE Request error: %v", err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		ctx.Log(fmt.Sprintf("SSE Unexpected response status: %d", resp.StatusCode))
		return
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-s.stop:
			return
		default:
		}
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			s.mutex.Lock()
			s.events = append(s.events, strings.TrimPrefix(line, "data: "))
			s.mutex.Unlock()
		}
	}
	if sErr := scanner.Err(); sErr != nil {
		ctx.Log(fmt.Sprintf("SSE Error reading stream: %v", sErr))
	}
}
