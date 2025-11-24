package kafka

import (
	"bytes"
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/with"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestCapturesAndListeners(t *testing.T) {
	do := &dummyDo{
		status: http.StatusOK,
		body:   []byte(`{"foo":"bar"}`),
	}
	options := Options{
		Subscribers: Subscribers{
			"topic_foo": {
				MaxMessages:  3,
				JsonMessages: true,
			},
			"topic_bar": {
				MaxMessages: -1,
			},
			"topic_baz": {
				MaxMessages: 1,
				Unmarshaler: func(msg Message) any {
					return msg
				},
			},
		},
		Wait:                time.Second * 3,
		InitialOffsetOldest: true,
	}
	endpoint := marrow.Endpoint("/api", "",
		marrow.Method("GET", "").AssertOK().
			Do(Publish(marrow.Before, "topic_foo", "foo", "bar1")).
			Do(Publish(marrow.Before, "topic_foo", "foo", "bar2")).
			Do(Publish(marrow.Before, "topic_foo", "foo", "bar3")).
			Do(Publish(marrow.Before, "topic_foo", "foo", "bar4")).
			Do(Publish(marrow.Before, "topic_foo", "foo", "bar5")).
			Do(Publish(marrow.Before, "topic_bar", "foo", "bar6")).
			Do(Publish(marrow.Before, "topic_baz", "foo", "bar7")).
			Wait(marrow.Before, 1000).
			AssertEqual(5, ReceivedMessages("topic_foo")).
			AssertEqual("bar4", marrow.JsonPath(ReceivedMessage("topic_foo", -2), "value")).
			AssertEqual(1, ReceivedMessages("topic_bar")).
			Wait(marrow.After, 1000).
			Do(Publish(marrow.After, "topic_bar", "foo", []byte("bar8"))).
			Do(Publish(marrow.After, "topic_bar", "foo", []any{"bar9"})).
			Do(Publish(marrow.After, "topic_bar", "foo", marrow.JSON{"foo": "bar10"})).
			Do(Publish(marrow.After, "topic_bar", "foo", []int{42, 43})).
			Do(Publish(marrow.After, "topic_bar", "foo", 42)).
			CaptureFunc(marrow.After, func(ctx marrow.Context) error {
				_, _ = marrow.ResolveValue(ReceivedMessages("not_listened_topic"), ctx)
				return nil
			}).
			CaptureFunc(marrow.After, func(ctx marrow.Context) error {
				_, _ = marrow.ResolveValue(ReceivedMessage("not_listened_topic", 0), ctx)
				return nil
			}).
			Do(Publish(marrow.After, "topic_foo", "", "", "not-kafka")),
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
	assert.Len(t, cov.Skipped, 0)
}

func TestPublish(t *testing.T) {
	w := Publish(marrow.After, "topic_foo", "bar", "baz")
	assert.Equal(t, marrow.After, w.When())
	assert.NotNil(t, w.Frame())
}

func TestReceivedMessages(t *testing.T) {
	c := ReceivedMessages("topic_foo")
	assert.Equal(t, "kafka.ReceivedMessages(\"topic_foo\")", fmt.Sprintf("%s", c))
}

func TestReceivedMessage(t *testing.T) {
	c := ReceivedMessage("topic_foo", 0)
	assert.Equal(t, "kafka.ReceivedMessage(\"topic_foo\", 0)", fmt.Sprintf("%s", c))
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
