package localstack

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-andiamo/marrow/with"
	"reflect"
	"strings"
)

type SecretsManagerService interface {
	Client() *secretsmanager.Client
	SecretARN(name string) (string, bool)
	Secret(name string) (string, bool)
	SecretBinary(name string) ([]byte, bool)
	setArn(name string, arn string)
}

type secretsManagerImage struct {
	options    Options
	host       string
	mappedPort string
	client     *secretsmanager.Client
	arns       map[string]string
}

var _ with.Image = (*secretsManagerImage)(nil)
var _ with.ImageResolveEnv = (*secretsManagerImage)(nil)
var _ SecretsManagerService = (*secretsManagerImage)(nil)

func (i *image) createSecretsManagerImage(ctx context.Context, awsCfg aws.Config) (err error) {
	img := &secretsManagerImage{
		options:    i.options,
		host:       i.host,
		mappedPort: i.mappedPort,
		client: secretsmanager.NewFromConfig(awsCfg,
			func(o *secretsmanager.Options) {
				o.BaseEndpoint = i.baseEndpoint()
				o.EndpointResolverV2 = secretsmanager.NewDefaultEndpointResolverV2()
			},
		),
		arns: make(map[string]string),
	}
	err = img.createSecrets(ctx)
	if err == nil {
		i.services[SecretsManager] = img
	}
	return err
}

func (s *secretsManagerImage) createSecrets(ctx context.Context) error {
	for k, v := range s.options.SecretsManager.Secrets {
		if out, err := s.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         aws.String(k),
			SecretString: aws.String(v),
		}); err == nil {
			s.arns[k] = *out.ARN
		} else {
			return err
		}
	}
	for k, v := range s.options.SecretsManager.JsonSecrets {
		var av []byte
		switch vt := v.(type) {
		case string:
			av = []byte(vt)
		case []byte:
			av = vt
		default:
			if v != nil {
				to := reflect.ValueOf(v)
				if to.Kind() == reflect.Map || to.Kind() == reflect.Slice || to.Kind() == reflect.Struct {
					av, _ = json.Marshal(v)
				}
			} else {
				return fmt.Errorf("invalid json secret type: %T", v)
			}
		}
		if out, err := s.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
			Name:         aws.String(k),
			SecretBinary: av,
		}); err == nil {
			s.arns[k] = *out.ARN
		} else {
			return err
		}
	}
	return nil
}

func (s *secretsManagerImage) Client() *secretsmanager.Client {
	return s.client
}

const secretsServiceImageName = "secrets-service"

func (s *secretsManagerImage) Name() string {
	return secretsServiceImageName
}

func (s *secretsManagerImage) Host() string {
	return s.host
}

func (s *secretsManagerImage) Port() string {
	return defaultPort
}

func (s *secretsManagerImage) MappedPort() string {
	return s.mappedPort
}

func (s *secretsManagerImage) IsDocker() bool {
	return true
}

func (s *secretsManagerImage) Username() string {
	return ""
}

func (s *secretsManagerImage) Password() string {
	return ""
}

func (s *secretsManagerImage) ResolveEnv(tokens ...string) (string, bool) {
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
				return s.SecretARN(tokens[1])
			}
		case "value":
			if len(tokens) > 1 {
				return s.Secret(tokens[1])
			}
		}
	}
	return "", false
}

func (s *secretsManagerImage) SecretARN(name string) (string, bool) {
	v, ok := s.arns[name]
	return v, ok
}

func (s *secretsManagerImage) Secret(name string) (string, bool) {
	if arn, ok := s.arns[name]; ok {
		if out, err := s.client.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{SecretId: aws.String(arn)}); err == nil {
			if out.SecretString != nil {
				return *out.SecretString, true
			}
		}
	}
	return "", false
}

func (s *secretsManagerImage) SecretBinary(name string) ([]byte, bool) {
	if arn, ok := s.arns[name]; ok {
		if out, err := s.client.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{SecretId: aws.String(arn)}); err == nil {
			if out.SecretString == nil {
				return out.SecretBinary, true
			}
		}
	}
	return nil, false
}

func (s *secretsManagerImage) setArn(name string, arn string) {
	s.arns[name] = arn
}

// SecretSet can be used as a before/after on marrow.Method .Capture
// and sets a secret in SecretsManager
//
// the value is resolved - if it's a string the secret is set to that string, otherwise the secret is stored as binary
//
// Note: setting a secret with a name already set will cause an error
//
//go:noinline
func SecretSet(when marrow.When, name string, value any, imgName ...string) marrow.BeforeAfter {
	return &capture[SecretsManagerService]{
		name:     fmt.Sprintf("SecretSet(%q)", name),
		when:     when,
		imgName:  imgName,
		defImage: secretsServiceImageName,
		run: func(ctx marrow.Context, img SecretsManagerService) (err error) {
			var av any
			if av, err = marrow.ResolveValue(value, ctx); err == nil {
				var secretString *string
				var secretBytes []byte
				switch avt := av.(type) {
				case string:
					secretString = aws.String(avt)
				case []byte:
					secretBytes = avt
				default:
					if av != nil {
						to := reflect.ValueOf(av)
						if to.Kind() == reflect.Map || to.Kind() == reflect.Slice || to.Kind() == reflect.Struct {
							secretBytes, _ = json.Marshal(av)
						} else {
							secretBytes = []byte(fmt.Sprintf("%v", av))
						}
					} else {
						s := ""
						secretString = &s
					}
				}
				var out *secretsmanager.CreateSecretOutput
				if out, err = img.Client().CreateSecret(context.Background(), &secretsmanager.CreateSecretInput{
					Name:         aws.String(name),
					SecretString: secretString,
					SecretBinary: secretBytes,
				}); err == nil {
					img.setArn(name, *out.ARN)
				}
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// SecretGet can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the value of the named secret in SecretManager
//
//go:noinline
func SecretGet(name string, imgName ...string) marrow.Resolvable {
	return &resolvable[SecretsManagerService]{
		name:     fmt.Sprintf("SecretGet(%q)", name),
		defImage: secretsServiceImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img SecretsManagerService) (result any, err error) {
			if arn, ok := img.SecretARN(name); ok {
				var out *secretsmanager.GetSecretValueOutput
				if out, err = img.Client().GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{SecretId: aws.String(arn)}); err == nil {
					if out.SecretString != nil {
						result = *out.SecretString
					} else {
						result = out.SecretBinary
					}
				}
			} else {
				err = fmt.Errorf("arn for secret %q not found", name)
			}
			return result, err
		},
		frame: framing.NewFrame(0),
	}
}
