package dragonfly

import (
	"bytes"
	"encoding/json"
	"fmt"
	. "github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/with"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
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
	endpoint := Endpoint("/api", "",
		Method("GET", "").AssertOK().
			Do(TopicListener("", "topic_foo", Receiver{})).
			Do(QueueListener("", "queue_extra", Receiver{})).
			Do(SendMessage(Before, "some_queue", "")).
			Do(SendMessage(Before, "queue_foo", `{"foo":"bar1"}`)).
			Do(SendMessage(Before, "queue_bar", `{"foo":"bar2"}`)).
			Do(SendMessage(Before, "queue_bar", `{"foo":"bar3"}`)).
			Do(SendMessage(Before, "queue_bar", `{"foo":"bar4"}`)).
			Do(SendMessage(Before, "queue_bar", `{"foo":"bar5"}`)).
			Do(SendMessage(Before, "queue_bar", "")).
			Do(SendMessage(Before, "queue_baz", `{"foo":"bar6"}`)).
			Do(PublishMessage(Before, "topic_foo", `{"foo":"bar7"}`)).
			Do(SendMessage(Before, "queue_extra", `{"foo":"bar8"}`)).
			Do(SetKey(Before, "foo", JSON{"foo": "bar"}, 0)).
			Wait(Before, 1000).
			AssertEqual(KeyExists("foo"), true).
			AssertEqual(Key("foo"), `{"foo":"bar"}`).
			AssertEqual(KeyExists("bar"), false).
			Do(SetKey(After, "bar", "kv", 0)).
			AssertEqual(Key("bar"), "kv").
			Do(SetKey(After, "bar", []byte("kv2"), 0)).
			AssertEqual(Key("bar"), "kv2").
			Do(SetKey(After, "bar", []any{"foo", "bar"}, 0)).
			AssertEqual(Key("bar"), `["foo","bar"]`).
			Do(SetKey(After, "bar", 42, 0)).
			AssertEqual(Key("bar"), "42").
			Do(SetKey(After, "bar", []int{42, 43}, 0)).
			AssertEqual(Key("bar"), "[42,43]").
			Do(DeleteKey(After, "bar")).
			AssertEqual(KeyExists("bar"), false).
			AssertEqual(QueueLen("some_queue"), 1).
			AssertEqual(1, ReceivedQueueMessages("queue_foo")).
			AssertEqual(5, ReceivedQueueMessages("queue_bar")).
			AssertEqual("bar5", JsonPath(ReceivedQueueMessage("queue_bar", -2), "foo")).
			AssertEqual(1, ReceivedTopicMessages("topic_foo")).
			AssertEqual("bar7", JsonPath(ReceivedTopicMessage("topic_foo", 0), "foo")).
			AssertEqual(1, EventsCount("topic_foo")).
			AssertEqual(`{"foo":"bar7"}`, First(Events("topic_foo"))).
			AssertEqual(1, EventsCount("queue_extra")).
			AssertEqual(`{"foo":"bar8"}`, First(Events("queue_extra"))).
			Wait(After, 1000).
			CaptureFunc(After, func(ctx Context) error {
				_, _ = ResolveValue(ReceivedQueueMessages("not_listened_queue"), ctx)
				return nil
			}).
			CaptureFunc(After, func(ctx Context) error {
				_, _ = ResolveValue(ReceivedTopicMessages("not_listened_topic"), ctx)
				return nil
			}).
			CaptureFunc(After, func(ctx Context) error {
				_, _ = ResolveValue(ReceivedQueueMessage("queue_bar", 10), ctx)
				return nil
			}).
			CaptureFunc(After, func(ctx Context) error {
				_, _ = ResolveValue(ReceivedQueueMessage("not_listened_queue", 0), ctx)
				return nil
			}).
			CaptureFunc(After, func(ctx Context) error {
				_, _ = ResolveValue(ReceivedTopicMessage("not_listened_topic", 0), ctx)
				return nil
			}).
			Do(PublishMessage(After, "topic_foo", "", "not-dragonfly")),
	)
	var cov *coverage.Coverage
	s := Suite(endpoint).Init(
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
	assert.Equal(t, "dragonfly.QueueLen(\"queue_foo\")", fmt.Sprintf("%s", c))
}

func TestSendMessage(t *testing.T) {
	w := SendMessage(After, "queue_foo", "bar")
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())
}

func TestPublishMessage(t *testing.T) {
	w := PublishMessage(After, "topic_foo", "bar")
	assert.Equal(t, After, w.When())
	assert.NotNil(t, w.Frame())
}

func TestReceivedQueueMessages(t *testing.T) {
	c := ReceivedQueueMessages("queue_foo")
	assert.Equal(t, "dragonfly.ReceivedQueueMessages(\"queue_foo\")", fmt.Sprintf("%s", c))
}

func TestReceivedTopicMessages(t *testing.T) {
	c := ReceivedTopicMessages("topic_foo")
	assert.Equal(t, "dragonfly.ReceivedTopicMessages(\"topic_foo\")", fmt.Sprintf("%s", c))
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
