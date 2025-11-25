package artemis

import (
	"bytes"
	"encoding/json"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/coverage"
	"github.com/go-andiamo/marrow/with"
	"github.com/go-stomp/stomp/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestResolvablesAndBeforeAfters(t *testing.T) {
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
				Unmarshaler: func(msg *stomp.Message) any {
					var jv any
					if err := json.Unmarshal(msg.Body, &jv); err == nil {
						return jv
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
			Do(SendMessage(marrow.Before, "some_queue", "")).
			Do(SendMessage(marrow.Before, "queue_foo", `{"foo":"bar1"}`)).
			Do(SendMessage(marrow.Before, "queue_bar", `{"foo":"bar2"}`)).
			Do(SendMessage(marrow.Before, "queue_bar", `{"foo":"bar3"}`)).
			Do(SendMessage(marrow.Before, "queue_bar", `{"foo":"bar4"}`)).
			Do(SendMessage(marrow.Before, "queue_bar", `{"foo":"bar5"}`)).
			Do(SendMessage(marrow.Before, "queue_bar", "")).
			Do(SendMessage(marrow.Before, "queue_baz", `{"foo":"bar6"}`)).
			Do(PublishMessage(marrow.Before, "topic_foo", `{"foo":"bar7"}`)).
			Wait(marrow.Before, 1000).
			AssertEqual(1, ReceivedQueueMessages("queue_foo")).
			AssertEqual(5, ReceivedQueueMessages("queue_bar")).
			AssertEqual("bar5", marrow.JsonPath(ReceivedQueueMessage("queue_bar", -2), "foo")).
			AssertEqual(1, ReceivedTopicMessages("topic_foo")).
			AssertEqual("bar7", marrow.JsonPath(ReceivedTopicMessage("topic_foo", 0), "foo")),
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
	assert.Len(t, cov.Failures, 0)
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
