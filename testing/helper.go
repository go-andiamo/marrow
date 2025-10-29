package testing

import (
	"context"
	"fmt"
	"github.com/go-andiamo/marrow/framing"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

type Helper interface {
	Run(name string, f func(t Helper)) bool
	End()
	Fail()
	Failed() bool
	FailNow()
	Log(args ...any)
	Error(args ...any)
	Fatal(args ...any)
	Context() context.Context
}

//go:noinline
func NewHelper(t *testing.T, stdout io.Writer, stderr io.Writer) Helper {
	f := framing.NewFrame(1)
	result := &helper{
		wrapped: t,
		name:    f.Name,
		frame:   f,
		start:   time.Now(),
		stdout:  stdout,
		stderr:  stderr,
	}
	if t == nil {
		if result.stdout == nil {
			result.stdout = os.Stdout
		}
		if result.stderr == nil {
			result.stderr = os.Stderr
		}
		result.logStart()
	}
	return result
}

type helper struct {
	parent   *helper
	wrapped  *testing.T
	name     string
	failed   bool
	stopped  bool
	start    time.Time
	duration time.Duration
	mu       sync.RWMutex
	frame    *framing.Frame
	stdout   io.Writer
	stderr   io.Writer
}

var _ Helper = (*helper)(nil)

func (h *helper) Run(name string, fn func(t Helper)) bool {
	if h.wrapped == nil {
		if h.stopped {
			return false
		}
		ch := &helper{
			parent: h,
			name:   name,
			frame:  h.frame,
			stdout: h.stdout,
			stderr: h.stderr,
		}
		ch.logStart()
		ch.start = time.Now()
		fn(ch)
		ch.duration = time.Since(ch.start)
		ch.logEnd()
		return ch.Failed()
	}
	return h.wrapped.Run(name, func(t *testing.T) {
		ch := &helper{
			parent:  h,
			wrapped: t,
			name:    name,
			frame:   h.frame,
		}
		fn(ch)
	})
}

func (h *helper) End() {
	if h.wrapped == nil && h.parent == nil {
		h.duration = time.Since(h.start)
		h.logEnd()
	}
}

func (h *helper) Fail() {
	if h.wrapped == nil {
		if h.parent != nil {
			h.parent.Fail()
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		h.failed = true
	} else {
		h.wrapped.Fail()
	}
}

func (h *helper) Failed() bool {
	if h.wrapped == nil {
		h.mu.RLock()
		defer h.mu.RUnlock()
		return h.failed
	} else {
		return h.wrapped.Failed()
	}
}

func (h *helper) FailNow() {
	if h.wrapped == nil {
		if h.parent != nil {
			h.parent.FailNow()
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		h.failed = true
		h.stopped = true
	} else {
		h.wrapped.FailNow()
	}
}

func (h *helper) Log(args ...any) {
	if h.wrapped == nil {
		h.log(fmt.Sprintln(args...))
	} else {
		h.wrapped.Log(args...)
	}
}

func (h *helper) Error(args ...any) {
	if h.wrapped == nil {
		h.log(fmt.Sprintln(args...))
		h.Fail()
	} else {
		h.wrapped.Error(args...)
	}
}

func (h *helper) Fatal(args ...any) {
	if h.wrapped == nil {
		h.log(fmt.Sprintln(args...))
		h.FailNow()
	} else {
		h.wrapped.Fatal(args...)
	}
}

func (h *helper) Context() context.Context {
	if h.wrapped == nil {
		return context.Background()
	} else {
		return h.wrapped.Context()
	}
}

func (h *helper) displayName() string {
	if h.parent != nil {
		return h.parent.displayName() + "/" + h.name
	}
	return h.name
}

func (h *helper) logStart() {
	_, _ = fmt.Fprintln(h.stdout, "=== RUN   "+h.displayName())
}

func (h *helper) logEnd() {
	if h.failed {
		_, _ = fmt.Fprintf(h.stderr, "\n--- FAIL: %s (%s)\n", h.displayName(), h.duration)
	} else {
		_, _ = fmt.Fprintf(h.stdout, "--- PASS: %s (%s)\n", h.displayName(), h.duration)
	}
}

func (h *helper) log(s string) {
	lines := strings.Split(s, "\n")
	lastSlash := strings.LastIndex(h.frame.File, "/")
	_, _ = fmt.Fprintf(h.stdout, "    %s:%d: %s\n", h.frame.File[lastSlash+1:], h.frame.Line, lines[0])
	for l := 1; l < len(lines); l++ {
		if line := lines[l]; line != "" {
			_, _ = fmt.Fprintf(h.stdout, "        %s\n", line)
		}
	}
}
