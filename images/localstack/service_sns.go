package localstack

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/go-andiamo/marrow/with"
	"strings"
)

type snsImage struct {
	options    Options
	host       string
	mappedPort string
	client     *sns.Client
	arns       map[string]string
}

var _ with.Image = (*snsImage)(nil)
var _ with.ImageResolveEnv = (*snsImage)(nil)

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
	err = img.createTopics(ctx)
	if err == nil {
		i.services[SNS] = img
	}
	return err
}

func (s *snsImage) createTopics(ctx context.Context) error {
	for _, topic := range s.options.SNS.CreateTopics {
		if out, err := s.client.CreateTopic(ctx, &topic); err == nil {
			s.arns[*topic.Name] = *out.TopicArn
		} else {
			return err
		}
	}
	return nil
}

const snsImageName = "sns"

func (s *snsImage) Name() string {
	return snsImageName
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
