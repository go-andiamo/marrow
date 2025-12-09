package localstack

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-andiamo/marrow/with"
	"strings"
)

type SSMService interface {
	Client() *ssm.Client
	PutParameter(name string, value string) error
}

type ssmImage struct {
	options    Options
	host       string
	mappedPort string
	client     *ssm.Client
}

var _ with.Image = (*ssmImage)(nil)
var _ with.ImageResolveEnv = (*ssmImage)(nil)
var _ SSMService = (*ssmImage)(nil)
var _ marrow.ImageStartupInitializer = (*ssmImage)(nil)

func (i *image) createSSMImage(ctx context.Context, awsCfg aws.Config) (err error) {
	img := &ssmImage{
		options:    i.options,
		host:       i.host,
		mappedPort: i.mappedPort,
		client: ssm.NewFromConfig(awsCfg,
			func(o *ssm.Options) {
				o.BaseEndpoint = i.baseEndpoint()
				o.EndpointResolverV2 = ssm.NewDefaultEndpointResolverV2()
			},
		),
	}
	i.services[SSM] = img
	return err
}

func (s *ssmImage) StartupInit(ctx marrow.Context) (err error) {
	var av any
	if av, err = marrow.ResolveValue(s.options.SSM.InitialParams, ctx); err == nil {
		mv := av.(map[string]any)
		for k, v := range mv {
			name := k
			if s.options.SSM.Prefix != "" {
				name = s.options.SSM.Prefix + "/" + name
			}
			if _, err = s.client.PutParameter(context.Background(), &ssm.PutParameterInput{
				Name:      aws.String(name),
				Value:     aws.String(fmt.Sprintf("%v", v)),
				Type:      types.ParameterTypeString,
				Overwrite: aws.Bool(true),
			}); err != nil {
				err = fmt.Errorf("failed to set SSM parameter %q: %w", name, err)
				break
			}
		}
	}
	return err
}

func (s *ssmImage) Client() *ssm.Client {
	return s.client
}

const SSMImageName = "ssm"

func (s *ssmImage) Name() string {
	return SSMImageName
}

func (s *ssmImage) Host() string {
	return s.host
}

func (s *ssmImage) Port() string {
	return defaultPort
}

func (s *ssmImage) MappedPort() string {
	return s.mappedPort
}

func (s *ssmImage) IsDocker() bool {
	return true
}

func (s *ssmImage) Username() string {
	return ""
}

func (s *ssmImage) Password() string {
	return ""
}

func (s *ssmImage) ResolveEnv(tokens ...string) (string, bool) {
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

func (s *ssmImage) PutParameter(name string, value string) error {
	if _, err := s.client.PutParameter(context.Background(), &ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      types.ParameterTypeString,
		Overwrite: aws.Bool(true),
	}); err != nil {
		return fmt.Errorf("failed to set parameter %q: %w", name, err)
	}
	return nil
}

// SSMPutParameter can be used as a before/after on marrow.Method .Capture
// and puts an SSM (System Manager) parameter
//
// note: the prefix from SSMOptions is not used for the name
//
//go:noinline
func SSMPutParameter(when marrow.When, name any, value any, imgName ...string) marrow.BeforeAfter {
	return &capture[SSMService]{
		name:     fmt.Sprintf("SSMPutParameter(%q)", name),
		when:     when,
		defImage: SSMImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img SSMService) (err error) {
			var nv any
			var vv any
			if nv, vv, err = marrow.ResolveValues(name, value, ctx); err == nil {
				err = img.PutParameter(fmt.Sprintf("%v", nv), fmt.Sprintf("%v", vv))
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}
