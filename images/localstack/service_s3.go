package localstack

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-andiamo/marrow/with"
	"strings"
)

type s3Image struct {
	options    Options
	host       string
	mappedPort string
	client     *s3.Client
}

var _ with.Image = (*s3Image)(nil)
var _ with.ImageResolveEnv = (*s3Image)(nil)

func (i *image) createS3Image(ctx context.Context, awsCfg aws.Config) (err error) {
	img := &s3Image{
		options:    i.options,
		host:       i.host,
		mappedPort: i.mappedPort,
		client: s3.NewFromConfig(awsCfg,
			func(o *s3.Options) {
				o.BaseEndpoint = i.baseEndpoint()
				o.EndpointResolverV2 = s3.NewDefaultEndpointResolverV2()
				o.UsePathStyle = true
			},
		),
	}
	err = img.createBuckets(ctx)
	if err == nil {
		i.services[S3] = img
	}
	return err
}

func (s *s3Image) createBuckets(ctx context.Context) error {
	for _, bucket := range s.options.S3.CreateBuckets {
		if _, err := s.client.CreateBucket(ctx, &bucket); err != nil {
			return err
		}
	}
	return nil
}

const s3ImageName = "s3"

func (s *s3Image) Name() string {
	return s3ImageName
}

func (s *s3Image) Host() string {
	return s.host
}

func (s *s3Image) Port() string {
	return defaultPort
}

func (s *s3Image) MappedPort() string {
	return s.mappedPort
}

func (s *s3Image) IsDocker() bool {
	return true
}

func (s *s3Image) Username() string {
	return ""
}

func (s *s3Image) Password() string {
	return ""
}

func (s *s3Image) ResolveEnv(tokens ...string) (string, bool) {
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
		}
	}
	return "", false
}
