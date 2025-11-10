package redis7

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/with"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapturesAndListeners(t *testing.T) {
	do := &dummyDo{
		status: http.StatusOK,
		body:   []byte(`{"foo":"bar"}`),
	}
	options := Options{
		Consumers: Receivers{
			"queue_foo": {
				MaxMessages: -1,
			},
			"queue_bar": {
				MaxMessages:  3,
				JsonMessages: true,
			},
			"queue_baz": {
				MaxMessages: 1,
				Unmarshaler: func(msg string) any {
					if msg != "" {
						var jv any
						if err := json.Unmarshal([]byte(msg), &jv); err == nil {
							return jv
						}
					}
					return nil
				},
			},
		},
		Subscribers: Receivers{
			"topic_foo": {
				MaxMessages:  1,
				JsonMessages: true,
			},
			"topic_bar": {
				MaxMessages: -1,
			},
		},
	}
	endpoint := marrow.Endpoint("/api", "",
		marrow.Method("GET", "").AssertOK().
			Capture(SendMessage(marrow.Before, "some_queue", "")).
			Capture(SendMessage(marrow.Before, "queue_foo", `{"foo":"bar1"}`)).
			Capture(SendMessage(marrow.Before, "queue_bar", `{"foo":"bar2"}`)).
			Capture(SendMessage(marrow.Before, "queue_bar", `{"foo":"bar3"}`)).
			Capture(SendMessage(marrow.Before, "queue_bar", `{"foo":"bar4"}`)).
			Capture(SendMessage(marrow.Before, "queue_bar", `{"foo":"bar5"}`)).
			Capture(SendMessage(marrow.Before, "queue_bar", "")).
			Capture(SendMessage(marrow.Before, "queue_baz", `{"foo":"bar6"}`)).
			Capture(PublishMessage(marrow.Before, "topic_foo", `{"foo":"bar7"}`)).
			Capture(SetKey(marrow.Before, "foo", marrow.JSON{"foo": "bar"}, 0)).
			Wait(marrow.Before, 1000).
			AssertEqual(KeyExists("foo"), true).
			AssertEqual(Key("foo"), `{"foo":"bar"}`).
			AssertEqual(KeyExists("bar"), false).
			Capture(SetKey(marrow.After, "bar", "kv", 0)).
			AssertEqual(Key("bar"), "kv").
			Capture(SetKey(marrow.After, "bar", []byte("kv2"), 0)).
			AssertEqual(Key("bar"), "kv2").
			Capture(SetKey(marrow.After, "bar", []any{"foo", "bar"}, 0)).
			AssertEqual(Key("bar"), `["foo","bar"]`).
			Capture(SetKey(marrow.After, "bar", 42, 0)).
			AssertEqual(Key("bar"), "42").
			Capture(SetKey(marrow.After, "bar", []int{42, 43}, 0)).
			AssertEqual(Key("bar"), "[42,43]").
			Capture(DeleteKey(marrow.After, "bar")).
			AssertEqual(KeyExists("bar"), false).
			AssertEqual(QueueLen("some_queue"), 1).
			AssertEqual(1, ReceivedQueueMessages("queue_foo")).
			AssertEqual(5, ReceivedQueueMessages("queue_bar")).
			AssertEqual("bar5", marrow.JsonPath(ReceivedQueueMessage("queue_bar", -2), "foo")).
			AssertEqual(1, ReceivedTopicMessages("topic_foo")).
			AssertEqual("bar7", marrow.JsonPath(ReceivedTopicMessage("topic_foo", 0), "foo")).
			Wait(marrow.After, 1000).
			CaptureFunc(marrow.After, func(ctx marrow.Context) error {
				_, _ = marrow.ResolveValue(ReceivedQueueMessages("not_listened_queue"), ctx)
				return nil
			}).
			CaptureFunc(marrow.After, func(ctx marrow.Context) error {
				_, _ = marrow.ResolveValue(ReceivedTopicMessages("not_listened_topic"), ctx)
				return nil
			}).
			CaptureFunc(marrow.After, func(ctx marrow.Context) error {
				_, _ = marrow.ResolveValue(ReceivedQueueMessage("queue_bar", 10), ctx)
				return nil
			}).
			CaptureFunc(marrow.After, func(ctx marrow.Context) error {
				_, _ = marrow.ResolveValue(ReceivedQueueMessage("not_listened_queue", 0), ctx)
				return nil
			}).
			CaptureFunc(marrow.After, func(ctx marrow.Context) error {
				_, _ = marrow.ResolveValue(ReceivedTopicMessage("not_listened_topic", 0), ctx)
				return nil
			}).
			Capture(PublishMessage(marrow.After, "topic_foo", "", "not-redis")),
	)
	var cov *coverage.Coverage
	s := marrow.Suite(endpoint).Init(
		With(options),
		with.HttpDo(do),
		with.ReportCoverage(func(coverage *coverage.Coverage) {
			cov = coverage
		}),
	)
	err := s.Run()
	require.NoError(t, err)
	assert.Len(t, cov.Failures, 1)
}

func TestQueueLen(t *testing.T) {
	c := QueueLen("queue_foo")
	assert.Equal(t, "redis.QueueLen(\"queue_foo\")", fmt.Sprintf("%s", c))
}

func TestSendMessage(t *testing.T) {
	w := SendMessage(marrow.After, "queue_foo", "bar")
	assert.Equal(t, marrow.After, w.When())
	assert.NotNil(t, w.Frame())
}

func TestPublishMessage(t *testing.T) {
	w := PublishMessage(marrow.After, "topic_foo", "bar")
	assert.Equal(t, marrow.After, w.When())
	assert.NotNil(t, w.Frame())
}

func TestReceivedQueueMessages(t *testing.T) {
	c := ReceivedQueueMessages("queue_foo")
	assert.Equal(t, "redis.ReceivedQueueMessages(\"queue_foo\")", fmt.Sprintf("%s", c))
}

func TestReceivedTopicMessages(t *testing.T) {
	c := ReceivedTopicMessages("topic_foo")
	assert.Equal(t, "redis.ReceivedTopicMessages(\"topic_foo\")", fmt.Sprintf("%s", c))
}

type dummyDo struct {
	status int
	body   []byte
	err    error
}

var _ common.HttpDo = (*dummyDo)(nil)

func (d *dummyDo) Do(req *http.Request) (*http.Response, error) {
	if d.err != nil {
		return nil, d.err
	}
	return &http.Response{
		StatusCode: d.status,
		Body:       io.NopCloser(bytes.NewReader(d.body)),
	}, nil
}
