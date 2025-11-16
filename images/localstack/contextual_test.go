package localstack

import (
	"bytes"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-andiamo/marrow"
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
	options := Options{
		Services: Services{All},
		Dynamo:   testDynamoOptions,
		S3: S3Options{
			CreateBuckets: []s3.CreateBucketInput{
				{
					Bucket: aws.String("my-bucket"),
				},
			},
		},
		SNS: SNSOptions{
			CreateTopics: []sns.CreateTopicInput{
				{
					Name: aws.String("my-topic"),
				},
			},
			TopicsSubscribe: true,
			MaxMessages:     2,
			JsonMessages:    true,
		},
		SQS: SQSOptions{
			CreateQueues: []sqs.CreateQueueInput{
				{
					QueueName: aws.String("my-queue"),
				},
			},
		},
	}
	endpoint := marrow.Endpoint("/api", "",
		marrow.Method("GET", "").AssertOK().
			Capture(SNSPublish(marrow.Before, "my-topic", 42)).
			Capture(SNSPublish(marrow.Before, "my-topic", &sns.PublishInput{Message: aws.String(`{"foo": "bar"}`)})).
			Capture(SNSPublish(marrow.Before, "my-topic", `{"foo": "bar"}`)).
			Capture(SNSPublish(marrow.Before, "my-topic", []byte(`{"foo": "bar"}`))).
			Capture(SNSPublish(marrow.Before, "my-topic", marrow.JSON{"foo": "bar"})).
			Capture(SQSSend(marrow.Before, "my-queue", marrow.JSON{"foo": "bar1"})).
			Capture(SQSPurge(marrow.Before, "my-queue")).
			Capture(SQSSend(marrow.Before, "my-queue", 42)).
			Capture(SQSSend(marrow.Before, "my-queue", `{"foo": "bar3"}`)).
			Capture(SQSSend(marrow.Before, "my-queue", []byte(`{"foo": "bar4"}`))).
			Capture(SQSSend(marrow.Before, "my-queue", &sqs.SendMessageInput{MessageBody: aws.String(`{"foo": "bar5"}`)})).
			Capture(SQSSend(marrow.Before, "my-queue", marrow.JSON{"foo": "bar6"})).
			SetVar(marrow.Before, "initial-count", DynamoItemsCount("TestTable")).
			Capture(DynamoPutItem(marrow.Before, "TestTable", marrow.JSON{"code": "foo", "value": "bar"})).
			SetVar(marrow.Before, "item", DynamoGetItem("TestTable", "code", "foo")).
			AssertEqual("bar", marrow.JsonPath(marrow.Var("item"), "value")).
			AssertEqual(marrow.Var("initial-count"), 0).
			AssertGreaterThan(DynamoItemsCount("TestTable"), marrow.Var("initial-count")).
			AssertEqual(5, SNSMessagesCount("")).
			AssertEqual(5, SNSMessagesCount("my-topic")).
			AssertEqual("bar", marrow.JsonTraverse(SNSMessages("my-topic"), -1, "Message.foo")).
			SetVar(marrow.After, "queue-msgs", SQSReceiveMessages("my-queue", 10, 0)).
			AssertEqual(5, marrow.JsonPath(marrow.Var("queue-msgs"), marrow.LEN)).
			AssertEqual(`{"foo":"bar6"}`, marrow.JsonTraverse(marrow.Var("queue-msgs"), -1, "Body")).
			Capture(DynamoDeleteItem(marrow.After, "TestTable", "code", "foo")),
		marrow.Method("GET", "again").AssertOK().
			Capture(S3CreateBucket(marrow.Before, "foo-bucket")).
			AssertEqual(0, DynamoItemsCount("TestTable")).
			AssertEqual(0, S3ObjectsCount("my-bucket", "")).
			AssertEqual(0, S3ObjectsCount("foo-bucket", "")),
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
