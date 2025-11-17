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
		SecretsManager: SecretsManagerOptions{
			Secrets: map[string]string{
				"foo": "bar",
			},
			JsonSecrets: map[string]any{
				"db": map[string]any{
					"name":     "my-db",
					"user":     "my-user",
					"password": "my-password",
				},
				"foo2": "bar2",
				"foo3": []byte("bar3"),
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
			Capture(DynamoDeleteItem(After, "TestTable", "code", "foo")).
			Capture(SecretSet(After, "secret-1", "my-secret-1")).
			Capture(SecretSet(After, "secret-2", []byte(`{"value": "my-secret-2"}`))).
			Capture(SecretSet(After, "secret-3", map[string]any{"value": "my-secret-3"})).
			Capture(SecretSet(After, "secret-4", 42)).
			Capture(SecretSet(After, "secret-5", nil)).
			AssertEqual("my-secret-1", SecretGet("secret-1")).
			AssertEqual("my-secret-3", JsonPath(Jsonify(SecretGet("secret-3")), "value")).
			AssertEqual("my-db", JsonPath(Jsonify(SecretGet("db")), "name")),
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
