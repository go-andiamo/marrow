package localstack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-andiamo/marrow/with"
	"reflect"
	"strings"
)

type SNSService interface {
	Client() *sns.Client
	TopicARN(topic string) (string, bool)
	topicListener() *snsListener
}

type snsImage struct {
	options    Options
	host       string
	mappedPort string
	client     *sns.Client
	arns       map[string]string
	listener   *snsListener
}

var _ with.Image = (*snsImage)(nil)
var _ with.ImageResolveEnv = (*snsImage)(nil)
var _ SNSService = (*snsImage)(nil)

func (i *image) createSnsImage(ctx context.Context, awsCfg aws.Config) (err error) {
	img := &snsImage{
		options:    i.options,
		host:       i.host,
		mappedPort: i.mappedPort,
		client: sns.NewFromConfig(awsCfg,
			func(o *sns.Options) {
				o.BaseEndpoint = i.baseEndpoint()
			},
		),
		arns: make(map[string]string),
	}
	if err = img.createTopics(ctx); err == nil {
		err = img.createListener()
	}
	if err == nil {
		i.services[SNS] = img
	}
	return err
}

func (s *snsImage) createTopics(ctx context.Context) error {
	for _, topic := range s.options.SNS.CreateTopics {
		if _, ok := s.arns[*topic.Name]; !ok {
			if out, err := s.client.CreateTopic(ctx, &topic); err == nil {
				s.arns[*topic.Name] = *out.TopicArn
			} else {
				return err
			}
		}
	}
	return nil
}

func (s *snsImage) createListener() (err error) {
	if s.options.SNS.TopicsSubscribe && len(s.options.SNS.CreateTopics) > 0 {
		s.listener, err = newSnsListener(s.mappedPort, s.options.SNS, s.client, s.arns)
	}
	return err
}

func (s *snsImage) topicListener() *snsListener {
	return s.listener
}

func (s *snsImage) shutdown() {
	if s.listener != nil {
		s.listener.shutdown()
	}
}

func (s *snsImage) Client() *sns.Client {
	return s.client
}

const SNSImageName = "sns"

func (s *snsImage) Name() string {
	return SNSImageName
}

func (s *snsImage) Host() string {
	return s.host
}

func (s *snsImage) Port() string {
	return defaultPort
}

func (s *snsImage) MappedPort() string {
	return s.mappedPort
}

func (s *snsImage) IsDocker() bool {
	return true
}

func (s *snsImage) Username() string {
	return ""
}

func (s *snsImage) Password() string {
	return ""
}

func (s *snsImage) ResolveEnv(tokens ...string) (string, bool) {
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
		case "arn":
			if len(tokens) > 1 {
				v, ok := s.arns[tokens[1]]
				return v, ok
			}
		}
	}
	return "", false
}

func (s *snsImage) TopicARN(topic string) (string, bool) {
	arn, ok := s.arns[topic]
	return arn, ok
}

// SNSPublish can be used as a before/after on marrow.Method .Capture
// and publishes a message to an SNS topic
//
//go:noinline
func SNSPublish(when marrow.When, topic string, message any, imgName ...string) marrow.BeforeAfter {
	return &capture[SNSService]{
		name:     fmt.Sprintf("SNSPublish(%q)", topic),
		when:     when,
		imgName:  imgName,
		defImage: SNSImageName,
		run: func(ctx marrow.Context, img SNSService) (err error) {
			var am any
			if am, err = marrow.ResolveValue(message, ctx); err == nil {
				var pi *sns.PublishInput
				if pi, err = buildPublishInput(topic, am, img); err == nil {
					_, err = img.Client().Publish(context.Background(), pi)
				}
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

func buildPublishInput(topic string, msg any, img SNSService) (result *sns.PublishInput, err error) {
	if pi, ok := msg.(*sns.PublishInput); ok {
		result = pi
		if result.TopicArn == nil {
			if arn, ok := img.TopicARN(topic); ok {
				result.TopicArn = &arn
			} else {
				return nil, fmt.Errorf("unable to resolve arn for topic %q", topic)
			}
		}
	} else if arn, ok := img.TopicARN(topic); ok {
		result = &sns.PublishInput{
			TopicArn: &arn,
		}
		switch mt := msg.(type) {
		case string:
			result.Message = &mt
		case []byte:
			s := string(mt)
			result.Message = &s
		default:
			to := reflect.TypeOf(msg)
			if to.Kind() == reflect.Slice || to.Kind() == reflect.Map || to.Kind() == reflect.Struct {
				data, _ := json.Marshal(msg)
				s := string(data)
				result.Message = &s
			} else {
				s := fmt.Sprintf("%v", msg)
				result.Message = &s
			}
		}
	} else {
		return nil, fmt.Errorf("unable to resolve arn for topic %q", topic)
	}
	return result, err
}

// SNSMessagesCount can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the count of messages received on the specified topic
//
// the topic arg can be an empty string or "*" - which counts across all topics
//
// Note: you must have set SNSOptions.TopicsSubscribe, otherwise the resolve will return an error
//
//go:noinline
func SNSMessagesCount(topic string, imgName ...string) marrow.Resolvable {
	return &resolvable[SNSService]{
		name:     fmt.Sprintf("SNSMessagesCount(%q)", topic),
		defImage: SNSImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img SNSService) (result any, err error) {
			if l := img.topicListener(); l != nil {
				return l.messagesCount(topic), nil
			}
			return nil, errors.New("cannot count messages with no topic listener")
		},
		frame: framing.NewFrame(0),
	}
}

// SNSMessages can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the messages received on the specified topic
//
// Note: you must have set SNSOptions.TopicsSubscribe, otherwise the resolve will return an error
//
//go:noinline
func SNSMessages(topic string, imgName ...string) marrow.Resolvable {
	return &resolvable[SNSService]{
		name:     fmt.Sprintf("SNSMessages(%q)", topic),
		defImage: SNSImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img SNSService) (result any, err error) {
			if l := img.topicListener(); l != nil {
				return l.messages(topic), nil
			}
			return nil, errors.New("cannot retrieve messages with no topic listener")
		},
		frame: framing.NewFrame(0),
	}
}
