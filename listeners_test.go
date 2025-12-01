package marrow

import (
	"bytes"
	"fmt"
	htesting "github.com/go-andiamo/marrow/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestEvents(t *testing.T) {
	v := Events("foo")
	assert.Equal(t, "Events(foo)", fmt.Sprintf("%s", v))

	ctx := newTestContext(nil)
	_, err := ResolveValue(v, ctx)
	require.Error(t, err)

	mock := &mockListener{}
	ctx.RegisterListener("foo", mock)
	av, err := ResolveValue(v, ctx)
	require.NoError(t, err)
	assert.Empty(t, av)
	assert.Equal(t, []string{"events"}, mock.calls)
}

func TestEventsCount(t *testing.T) {
	v := EventsCount("foo")
	assert.Equal(t, "EventsCount(foo)", fmt.Sprintf("%s", v))

	ctx := newTestContext(nil)
	_, err := ResolveValue(v, ctx)
	require.Error(t, err)

	mock := &mockListener{}
	ctx.RegisterListener("foo", mock)
	av, err := ResolveValue(v, ctx)
	require.NoError(t, err)
	assert.Empty(t, av)
	assert.Equal(t, []string{"count"}, mock.calls)
}

func TestEventClear(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c := EventsClear("foo")
		assert.Equal(t, "EventsClear(foo)", c.Name())
		assert.NotNil(t, c.Frame())
	})
	t.Run("run", func(t *testing.T) {
		c := EventsClear("foo")
		ctx := newTestContext(nil)
		mock := &mockListener{}
		ctx.RegisterListener("foo", mock)
		err := c.Run(ctx)
		require.NoError(t, err)
		assert.Equal(t, []string{"clear"}, mock.calls)
	})
	t.Run("run errors with no listener", func(t *testing.T) {
		c := EventsClear("foo")
		ctx := newTestContext(nil)
		err := c.Run(ctx)
		require.Error(t, err)
	})
}

func TestSSEListener(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		v := SSEListener("foo", "/api/events")
		assert.Equal(t, Before, v.When())
		assert.NotNil(t, v.Frame())
		assert.Equal(t, "SSEListener(foo, /api/events)", fmt.Sprintf("%s", v))
	})
	t.Run("run", func(t *testing.T) {
		v := SSEListener("foo", "/api/events")
		ctx := newTestContext(nil)
		sse := &dummySse{
			status: 200,
			events: []string{"foo1", "foo2", "foo3"},
		}
		ctx.httpDo = sse
		err := v.Run(ctx)
		require.NoError(t, err)
		sse.wg.Wait()
		time.Sleep(time.Millisecond * 5)
		vl := v.(Listener)
		vl.Stop()
		assert.Len(t, vl.Events(), 3)
		assert.Equal(t, 3, vl.EventsCount())
		vl.Clear()
		assert.Empty(t, vl.Events())
	})
	t.Run("run twice", func(t *testing.T) {
		v := SSEListener("foo", "/api/events")
		ctx := newTestContext(nil)
		sse := &dummySse{
			status: 200,
			events: []string{"foo1", "foo2", "foo3"},
		}
		ctx.httpDo = sse
		err := v.Run(ctx)
		require.NoError(t, err)
		sse.wg.Wait()
		time.Sleep(time.Millisecond * 5)
		vl := v.(Listener)
		vl.Stop()
		assert.Len(t, vl.Events(), 3)
		assert.Equal(t, 3, vl.EventsCount())

		// again...
		err = v.Run(ctx)
		assert.Empty(t, vl.Events())
	})
	t.Run("run bad status", func(t *testing.T) {
		v := SSEListener("foo", "/api/events")
		ctx := newTestContext(nil)
		ctx.httpDo = &dummySse{
			status: 400,
		}
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)
		err := v.Run(ctx)
		require.NoError(t, err)
		time.Sleep(time.Millisecond * 5)
		vl := v.(Listener)
		vl.Stop()
		assert.Contains(t, buf.String(), "SSE Unexpected response status:")
	})
	t.Run("run errors", func(t *testing.T) {
		v := SSEListener("foo", "/api/events")
		ctx := newTestContext(nil)
		ctx.httpDo = &dummySse{
			err: fmt.Errorf("error"),
		}
		var buf bytes.Buffer
		ctx.testing = htesting.NewHelper(nil, &buf, &buf)
		err := v.Run(ctx)
		require.NoError(t, err)
		time.Sleep(time.Millisecond * 5)
		vl := v.(Listener)
		vl.Stop()
		assert.Contains(t, buf.String(), "SSE Request error:")
	})
	t.Run("run mismatch name/type", func(t *testing.T) {
		ctx := newTestContext(nil)
		ctx.RegisterListener("foo", &mockListener{})
		v := SSEListener("foo", "/api/events")
		err := v.Run(ctx)
		require.Error(t, err)
	})
}

type dummySse struct {
	status   int
	events   []string
	interval time.Duration
	hdrs     map[string]string
	err      error
	wg       sync.WaitGroup
}

func (d *dummySse) Do(req *http.Request) (*http.Response, error) {
	if d.err != nil {
		return nil, d.err
	}
	if d.status == 0 {
		d.status = http.StatusOK
	}
	d.wg.Add(len(d.events))
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		for i, ev := range d.events {
			if i > 0 && d.interval > 0 {
				time.Sleep(d.interval)
			}
			line := "data: " + ev + "\n\n"
			_, err := io.WriteString(pw, line)
			d.wg.Done()
			if err != nil {
				return
			}
		}
	}()
	resp := &http.Response{
		StatusCode: d.status,
		Status:     fmt.Sprintf("%d %s", d.status, http.StatusText(d.status)),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       pr,
		Request:    req,
	}
	for k, v := range d.hdrs {
		resp.Header.Set(k, v)
	}
	if resp.Header.Get("Content-Type") == "" {
		resp.Header.Set("Content-Type", "text/event-stream")
	}
	return resp, nil
}
