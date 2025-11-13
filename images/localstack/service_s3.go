package localstack

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-andiamo/marrow/with"
	"strings"
)

type S3Service interface {
	Client() *s3.Client
	CountObjects(bucket string, prefix string) (int, error)
	CreateBucket(bucket string) error
}

type s3Image struct {
	options    Options
	host       string
	mappedPort string
	client     *s3.Client
}

var _ with.Image = (*s3Image)(nil)
var _ with.ImageResolveEnv = (*s3Image)(nil)
var _ S3Service = (*s3Image)(nil)

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

func (s *s3Image) Client() *s3.Client {
	return s.client
}

func (s *s3Image) CountObjects(bucket string, prefix string) (int, error) {
	total := 0
	p := s3.NewListObjectVersionsPaginator(s.client, &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})
	for p.HasMorePages() {
		out, err := p.NextPage(context.Background())
		if err != nil {
			return 0, err
		}
		total += len(out.Versions)
	}
	return total, nil
}

func (s *s3Image) CreateBucket(bucket string) error {
	_, err := s.client.CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	})
	return err
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

// S3ObjectsCount can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the count of items in an S3 bucket
//
//go:noinline
func S3ObjectsCount(bucket string, prefix string, imgName ...string) marrow.Resolvable {
	return &resolvable[S3Service]{
		name:     fmt.Sprintf("S3ObjectsCount(%q, %q)", bucket, prefix),
		defImage: s3ImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img S3Service) (result any, err error) {
			return img.CountObjects(bucket, prefix)
		},
		frame: framing.NewFrame(0),
	}
}

// S3CreateBucket can be used as a before/after on marrow.Method .Capture
// and creates an S3 bucket
//
//go:noinline
func S3CreateBucket(when marrow.When, bucket string, imgName ...string) marrow.BeforeAfter {
	return &capture[S3Service]{
		name:     fmt.Sprintf("S3CreateBucket(%q)", bucket),
		when:     when,
		defImage: s3ImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img S3Service) error {
			return img.CreateBucket(bucket)
		},
		frame: framing.NewFrame(0),
	}
}
