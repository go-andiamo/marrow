package localstack

import (
	"bytes"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
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

func TestResolvablesAndBeforeAfters(t *testing.T) {
	do := &dummyDo{
		status: http.StatusOK,
		body:   []byte(`{"foo":"bar"}`),
	}
	const (
		testBucket = "my-bucket"
		testTopic  = "my-topic"
		testQueue  = "my-queue"
	)
	options := Options{
		Services: Services{All},
		Dynamo:   testDynamoOptions,
		S3: S3Options{
			CreateBuckets: []s3.CreateBucketInput{
				{
					Bucket: aws.String(testBucket),
				},
			},
		},
		SNS: SNSOptions{
			CreateTopics: []sns.CreateTopicInput{
				{
					Name: aws.String(testTopic),
				},
			},
			TopicsSubscribe: true,
			MaxMessages:     2,
			JsonMessages:    true,
		},
		SQS: SQSOptions{
			CreateQueues: []sqs.CreateQueueInput{
				{
					QueueName: aws.String(testQueue),
				},
			},
		},
	}
	const (
		varItem         = Var("item")
		varInitialCount = Var("initial-count")
		varQueueMsgs    = Var("queue-msgs")
		varLastQueueMsg = Var("last-queue-msg")
	)
	endpoint := Endpoint("/api", "",
		Method("GET", "").AssertOK().
			Capture(SNSPublish(Before, testTopic, 42)).
			Capture(SNSPublish(Before, testTopic, &sns.PublishInput{Message: aws.String(`{"foo": "bar"}`)})).
			Capture(SNSPublish(Before, testTopic, `{"foo": "bar"}`)).
			Capture(SNSPublish(Before, testTopic, []byte(`{"foo": "bar"}`))).
			Capture(SNSPublish(Before, testTopic, JSON{"foo": "bar"})).
			Capture(SQSSend(Before, testQueue, JSON{"foo": "bar1"})).
			Capture(SQSPurge(Before, testQueue)).
			Capture(SQSSend(Before, testQueue, 42)).
			Capture(SQSSend(Before, testQueue, `{"foo": "bar3"}`)).
			Capture(SQSSend(Before, testQueue, []byte(`{"foo": "bar4"}`))).
			Capture(SQSSend(Before, testQueue, &sqs.SendMessageInput{MessageBody: aws.String(`{"foo": "bar5"}`)})).
			Capture(SQSSend(Before, testQueue, JSON{"foo": "bar6"})).
			SetVar(Before, varInitialCount, DynamoItemsCount("TestTable")).
			Capture(DynamoPutItem(Before, "TestTable", JSON{"code": "foo", "value": "bar"})).
			SetVar(Before, varItem, DynamoGetItem("TestTable", "code", "foo")).
			AssertEqual("bar", JsonPath(varItem, "value")).
			AssertEqual(varInitialCount, 0).
			AssertGreaterThan(DynamoItemsCount("TestTable"), varInitialCount).
			AssertEqual(5, SNSMessagesCount("")).
			AssertEqual(5, SNSMessagesCount(testTopic)).
			AssertEqual("bar", JsonTraverse(SNSMessages(testTopic), -1, "Message.foo")).
			SetVar(After, varQueueMsgs, SQSReceiveMessages(testQueue, 10, 0)).
			AssertEqual(5, JsonPath(varQueueMsgs, LEN)).
			SetVar(After, varLastQueueMsg, Jsonify(JsonTraverse(varQueueMsgs, LAST, "Body"))).
			AssertEqual("bar6", JsonPath(varLastQueueMsg, "foo")).
			Capture(DynamoDeleteItem(After, "TestTable", "code", "foo")),
		Method("GET", "again").AssertOK().
			Capture(S3CreateBucket(Before, "foo-bucket")).
			AssertEqual(0, DynamoItemsCount("TestTable")).
			AssertEqual(0, S3ObjectsCount(testBucket, "")).
			AssertEqual(0, S3ObjectsCount("foo-bucket", "")),
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
