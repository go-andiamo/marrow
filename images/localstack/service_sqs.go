package localstack

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-andiamo/marrow/with"
	"reflect"
	"strings"
)

type SQSService interface {
	Client() *sqs.Client
	QueueURL(queue string) (string, bool)
}

type sqsImage struct {
	options    Options
	host       string
	mappedPort string
	client     *sqs.Client
	urls       map[string]string
}

var _ with.Image = (*sqsImage)(nil)
var _ with.ImageResolveEnv = (*sqsImage)(nil)
var _ SQSService = (*sqsImage)(nil)

func (i *image) createSqsImage(ctx context.Context, awsCfg aws.Config) (err error) {
	img := &sqsImage{
		options:    i.options,
		host:       i.host,
		mappedPort: i.mappedPort,
		client: sqs.NewFromConfig(awsCfg,
			func(o *sqs.Options) {
				o.BaseEndpoint = i.baseEndpoint()
			},
		),
		urls: make(map[string]string),
	}
	err = img.createQueues(ctx)
	if err == nil {
		i.services[SQS] = img
	}
	return err
}

func (s *sqsImage) createQueues(ctx context.Context) error {
	for _, queue := range s.options.SQS.CreateQueues {
		if _, ok := s.urls[*queue.QueueName]; !ok {
			if out, err := s.client.CreateQueue(ctx, &queue); err == nil {
				url := *out.QueueUrl
				s.urls[*queue.QueueName] = strings.Replace(url, "localhost:4566", "localhost:"+s.mappedPort, 1)
			} else {
				return err
			}
		}
	}
	return nil
}

func (s *sqsImage) Client() *sqs.Client {
	return s.client
}

const sqsImageName = "sqs"

func (s *sqsImage) Name() string {
	return sqsImageName
}

func (s *sqsImage) Host() string {
	return s.host
}

func (s *sqsImage) Port() string {
	return defaultPort
}

func (s *sqsImage) MappedPort() string {
	return s.mappedPort
}

func (s *sqsImage) IsDocker() bool {
	return true
}

func (s *sqsImage) Username() string {
	return ""
}

func (s *sqsImage) Password() string {
	return ""
}

func (s *sqsImage) ResolveEnv(tokens ...string) (string, bool) {
	if len(tokens) > 0 {
		switch strings.ToLower(tokens[0]) {
		case "region":
			return s.options.region(), true
		case "accesskey":
			return s.options.accessKey(), true
		case "secretkey":
			return s.options.secretKey(), true
		case "sessiontoken":
			return s.options.sessionToken(), true
		case "url":
			if len(tokens) > 1 {
				v, ok := s.urls[tokens[1]]
				return v, ok
			}
		}
	}
	return "", false
}

func (s *sqsImage) QueueURL(queue string) (string, bool) {
	url, ok := s.urls[queue]
	return url, ok
}

// SQSSend can be used as a before/after on marrow.Method .Capture
// and sends a message to an SQS queue
//
//go:noinline
func SQSSend(when marrow.When, queue string, message any, imgName ...string) marrow.BeforeAfter {
	return &capture[SQSService]{
		name:     fmt.Sprintf("SQSSend(%q)", queue),
		when:     when,
		imgName:  imgName,
		defImage: sqsImageName,
		run: func(ctx marrow.Context, img SQSService) (err error) {
			var am any
			if am, err = marrow.ResolveValue(message, ctx); err == nil {
				var smi *sqs.SendMessageInput
				if smi, err = buildSendMessageInput(queue, am, img); err == nil {
					_, err = img.Client().SendMessage(context.Background(), smi)
				}
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

func buildSendMessageInput(queue string, msg any, img SQSService) (result *sqs.SendMessageInput, err error) {
	if mi, ok := msg.(*sqs.SendMessageInput); ok {
		result = mi
		if result.QueueUrl == nil {
			if url, ok := img.QueueURL(queue); ok {
				result.QueueUrl = &url
			} else {
				return nil, fmt.Errorf("unable to resolve url for queue %q", queue)
			}
		}
	} else if url, ok := img.QueueURL(queue); ok {
		result = &sqs.SendMessageInput{
			QueueUrl: &url,
		}
		switch mt := msg.(type) {
		case string:
			result.MessageBody = &mt
		case []byte:
			s := string(mt)
			result.MessageBody = &s
		default:
			to := reflect.TypeOf(msg)
			if to.Kind() == reflect.Slice || to.Kind() == reflect.Map || to.Kind() == reflect.Struct {
				data, _ := json.Marshal(msg)
				s := string(data)
				result.MessageBody = &s
			} else {
				s := fmt.Sprintf("%v", msg)
				result.MessageBody = &s
			}
		}
	} else {
		return nil, fmt.Errorf("unable to resolve url for queue %q", queue)
	}
	return result, err
}

// SQSPurge can be used as a before/after on marrow.Method .Capture
// and purges all message on an SQS queue
//
//go:noinline
func SQSPurge(when marrow.When, queue string, imgName ...string) marrow.BeforeAfter {
	return &capture[SQSService]{
		name:     fmt.Sprintf("SQSPurge(%q)", queue),
		when:     when,
		imgName:  imgName,
		defImage: sqsImageName,
		run: func(ctx marrow.Context, img SQSService) (err error) {
			if url, ok := img.QueueURL(queue); ok {
				_, err = img.Client().PurgeQueue(context.Background(), &sqs.PurgeQueueInput{QueueUrl: &url})
			} else {
				return fmt.Errorf("unable to resolve url for queue %q", queue)
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// SQSReceiveMessages can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the messages received from the specified queue
//
// Note: the AWS client expects the maxMessages to not be greater than 10
//
//go:noinline
func SQSReceiveMessages(queue string, maxMessages int, wait int, imgName ...string) marrow.Resolvable {
	return &resolvable[SQSService]{
		name:     fmt.Sprintf("SQSReceiveMessages(%q)", queue),
		defImage: sqsImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img SQSService) (result any, err error) {
			if url, ok := img.QueueURL(queue); ok {
				var out *sqs.ReceiveMessageOutput
				in := &sqs.ReceiveMessageInput{
					QueueUrl:            &url,
					MaxNumberOfMessages: int32(maxMessages),
					WaitTimeSeconds:     int32(wait),
				}
				if out, err = img.Client().ReceiveMessage(context.Background(), in); err == nil {
					msgs := make([]any, len(out.Messages))
					for i, m := range out.Messages {
						msgs[i] = map[string]any{
							"MessageAttributes": m.MessageAttributes,
							"Body":              *m.Body,
							"MessageId":         *m.MessageId,
						}
					}
					result = msgs
				}
			} else {
				err = fmt.Errorf("unable to resolve url for queue %q", queue)
			}
			return result, err
		},
		frame: framing.NewFrame(0),
	}
}
