package nats

import (
	"bytes"
	. "github.com/go-andiamo/marrow"
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

func TestResolvablesAndBeforeAfters(t *testing.T) {
	myBucket := Var("my-bucket")
	bucketName := "test-bucket"
	mySubject := Var("my-subject")
	subject := "test.subject"
	do := &dummyDo{
		status: http.StatusOK,
		body:   []byte(`{"foo":"bar"}`),
	}
	opts := Options{
		CreateKeyValueBuckets: map[string]KeyValueBucket{
			bucketName: {
				TTL: 24 * time.Hour,
			},
		},
	}
	endpoint := Endpoint("/api", "",
		BucketWatch("bucket-watch", myBucket),
		Subscribe("subscription", mySubject),
		Method("GET", "").AssertOK().
			Do(Publish(Before, mySubject, JSON{"foo": "bar"})).
			Do(PutKey(Before, myBucket, "foo", "some-value")).
			Do(PutKey(Before, myBucket, "foo", "some-value2")).
			AssertEqual("some-value2", Key(myBucket, "foo")).
			AssertEqual(1, EventsCount("subscription")).
			SetVar(After, "last-msg", Last(Events("subscription"))).
			AssertEqual(`{"foo":"bar"}`, Var("last-msg")).
			AssertTrue(KeyExists(myBucket, "foo")).
			Do(DeleteKey(After, myBucket, "foo")).
			AssertFalse(KeyExists(myBucket, "foo")).
			AssertEqual(3, EventsCount("bucket-watch")).
			SetVar(After, "last-kv-event", Last(Events("bucket-watch"))).
			AssertEqual("foo", JsonPath(Var("last-kv-event"), "key")).
			AssertEqual("KeyValueDeleteOp", JsonPath(Var("last-kv-event"), "operation")).
			Capture(After, EventsClear("bucket-watch")).
			Capture(After, EventsClear("subscription")).
			AssertEqual(0, EventsCount("bucket-watch")).
			AssertEqual(0, EventsCount("subscription")),
	)
	var cov *coverage.Coverage
	s := Suite(endpoint).Init(
		With(opts),
		with.HttpDo(do),
		with.Var(string(myBucket), bucketName),
		with.Var(string(mySubject), subject),
		with.ReportCoverage(func(coverage *coverage.Coverage) {
			cov = coverage
		}),
	)
	err := s.Run()
	require.NoError(t, err)
	assert.Len(t, cov.Failures, 0)
	assert.Len(t, cov.Unmet, 0)
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
