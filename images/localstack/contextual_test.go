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
		Lambda: LambdaOptions{
			CreateFunctions: []string{"foo-func"},
		},
		SecretsManager: SecretsManagerOptions{
			Secrets: map[string]any{
				"foo":            "bar",
				"secret-topic-1": TemplateString("{$svc:sns:arn:" + testTopic + "}"),
			},
			JsonSecrets: map[string]any{
				"secret-topic-2": TemplateString("{$svc:sns:arn:" + testTopic + "}"),
				"db": map[string]any{
					"name":     "my-db",
					"user":     "my-user",
					"password": "my-password",
				},
				"foo2": "bar2",
				"foo3": []byte("bar3"),
				"foo4": map[string]any{
					"foo": "bar4",
				},
			},
		},
		SSM: SSMOptions{
			Prefix: "my-app/settings",
			InitialParams: map[string]any{
				"use-topic": TemplateString("{$svc:sns:arn:" + testTopic + "}"),
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
			Do(SSMPutParameter(Before, "foo", 42)).
			Do(SNSPublish(Before, testTopic, 42)).
			Do(SNSPublish(Before, testTopic, &sns.PublishInput{Message: aws.String(`{"foo": "bar"}`)})).
			Do(SNSPublish(Before, testTopic, `{"foo": "bar"}`)).
			Do(SNSPublish(Before, testTopic, []byte(`{"foo": "bar"}`))).
			Do(SNSPublish(Before, testTopic, JSON{"foo": "bar"})).
			Do(SQSSend(Before, testQueue, JSON{"foo": "bar1"})).
			Do(SQSPurge(Before, testQueue)).
			Do(SQSSend(Before, testQueue, 42)).
			Do(SQSSend(Before, testQueue, `{"foo": "bar3"}`)).
			Do(SQSSend(Before, testQueue, []byte(`{"foo": "bar4"}`))).
			Do(SQSSend(Before, testQueue, &sqs.SendMessageInput{MessageBody: aws.String(`{"foo": "bar5"}`)})).
			Do(SQSSend(Before, testQueue, JSON{"foo": "bar6"})).
			SetVar(Before, varInitialCount, DynamoItemsCount("TestTable")).
			Do(DynamoPutItem(Before, "TestTable", JSON{"code": "foo", "value": "bar"})).
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
			Do(DynamoDeleteItem(After, "TestTable", "code", "foo")).
			Do(SecretSet(After, "secret-1", "my-secret-1")).
			Do(SecretSet(After, "secret-2", []byte(`{"value": "my-secret-2"}`))).
			Do(SecretSet(After, "secret-3", map[string]any{"value": "my-secret-3"})).
			Do(SecretSet(After, "secret-4", 42)).
			Do(SecretSet(After, "secret-5", nil)).
			AssertEqual("my-secret-1", SecretGet("secret-1")).
			AssertEqual("my-secret-3", JsonPath(Jsonify(SecretGet("secret-3")), "value")).
			AssertEqual("my-db", JsonPath(Jsonify(SecretGet("db")), "name")).
			AssertEqual(0, LambdaInvokedCount("func-foo")).
			AssertNotEqual("", TemplateString("{$svc:secrets-service:arn:foo}")).
			AssertEqual("bar", TemplateString("{$svc:secrets-service:value:foo}")).
			AssertEqual(`{"foo":"bar4"}`, TemplateString("{$svc:secrets-service:value:foo4}")),
		Method("GET", "again").AssertOK().
			Do(S3CreateBucket(Before, "foo-bucket")).
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
