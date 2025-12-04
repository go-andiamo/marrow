package localstack

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	cwl "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-andiamo/marrow/with"
	"io"
	"strings"
	"time"
)

type LambdaService interface {
	Client() *lambda.Client
	InvokedCount(name string) (count int, err error)
}

type lambdaImage struct {
	options    Options
	host       string
	mappedPort string
	arns       map[string]string
	client     *lambda.Client
	cwlc       *cwl.Client
}

var _ with.Image = (*lambdaImage)(nil)
var _ with.ImageResolveEnv = (*lambdaImage)(nil)
var _ LambdaService = (*lambdaImage)(nil)

func (i *image) createLambdaImage(ctx context.Context, awsCfg aws.Config) (err error) {
	img := &lambdaImage{
		options:    i.options,
		host:       i.host,
		mappedPort: i.mappedPort,
		arns:       make(map[string]string),
		client: lambda.NewFromConfig(awsCfg,
			func(o *lambda.Options) {
				o.BaseEndpoint = i.baseEndpoint()
				o.EndpointResolverV2 = lambda.NewDefaultEndpointResolverV2()
			},
		),
		cwlc: i.cwlc,
	}
	err = img.createFunctions(ctx)
	if err == nil {
		i.services[Lambda] = img
	}
	return err
}

func (s *lambdaImage) createFunctions(ctx context.Context) (err error) {
	if len(s.options.Lambda.CreateFunctions) > 0 {
		var lzip []byte
		if lzip, err = buildLambdaZip(); err == nil {
			fm := make(map[string]struct{}, len(s.options.Lambda.CreateFunctions))
			for _, fn := range s.options.Lambda.CreateFunctions {
				fm[fn] = struct{}{}
			}
			first := true
			for fn := range fm {
				var out *lambda.CreateFunctionOutput
				inp := &lambda.CreateFunctionInput{
					FunctionName: aws.String(fn),
					Runtime:      "python3.14", //types.RuntimePython311,
					Handler:      aws.String("lambda_function.lambda_handler"),
					Role:         aws.String("arn:aws:iam::000000000000:role/lambda-execution-role"),
					Code: &types.FunctionCode{
						ZipFile: lzip,
					},
				}
				if out, err = s.client.CreateFunction(ctx, inp); err == nil {
					s.arns[fn] = *out.FunctionArn
					err = s.waitLambdaActive(ctx, fn, first)
					first = false
				}
				if err != nil {
					break
				}
			}
		}
	}
	return err
}

func (s *lambdaImage) waitLambdaActive(ctx context.Context, fn string, first bool) error {
	var timeout time.Duration
	if first {
		timeout = s.options.Lambda.PullTimeout
		if timeout <= 0 {
			timeout = 5 * time.Minute
		}
	} else {
		timeout = s.options.Lambda.ActiveTimeout
		if timeout <= 0 {
			timeout = time.Minute
		}
	}
	deadline := time.Now().Add(timeout)
	for {
		if time.Until(deadline) <= 0 {
			return fmt.Errorf("lambda %q did not become active within %s", fn, timeout)
		}
		out, err := s.client.GetFunctionConfiguration(ctx, &lambda.GetFunctionConfigurationInput{
			FunctionName: aws.String(fn),
		})
		if err != nil {
			return fmt.Errorf("lambda get configuration failed: %w", err)
		}
		switch out.State {
		case types.StateActive:
			return nil
		case types.StateFailed:
			reason := "unknown"
			if out.StateReason != nil {
				reason = *out.StateReason
			}
			return fmt.Errorf("lambda %s failed to deploy: %s (%s)", fn, string(out.State), reason)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func buildLambdaZip() ([]byte, error) {
	const code = `
def lambda_handler(event, context):
    print("lambda invoked with:", event)
    return {"ok": True}
`
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)
	var w io.Writer
	var err error
	if w, err = zw.Create("lambda_function.py"); err == nil {
		_, err = io.WriteString(w, code)
		if cerr := zw.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return buf.Bytes(), err
}

func (s *lambdaImage) Client() *lambda.Client {
	return s.client
}

func (s *lambdaImage) InvokedCount(name string) (count int, err error) {
	logGroup := "/aws/lambda/" + name
	var next *string
	var out *cwl.FilterLogEventsOutput
	for err == nil {
		if out, err = s.cwlc.FilterLogEvents(context.Background(), &cwl.FilterLogEventsInput{
			LogGroupName: &logGroup,
			NextToken:    next,
		}); err == nil {
			for _, event := range out.Events {
				if event.Message != nil && strings.Contains(*event.Message, "invoked") {
					count++
				}
			}
			if out.NextToken == nil {
				break
			}
			next = out.NextToken
		} else if strings.Contains(err.Error(), "log group does not exist") {
			return 0, nil
		}
	}
	return count, err
}

const lambdaImageName = "lambda"

func (s *lambdaImage) Name() string {
	return lambdaImageName
}

func (s *lambdaImage) Host() string {
	return s.host
}

func (s *lambdaImage) Port() string {
	return defaultPort
}

func (s *lambdaImage) MappedPort() string {
	return s.mappedPort
}

func (s *lambdaImage) IsDocker() bool {
	return true
}

func (s *lambdaImage) Username() string {
	return ""
}

func (s *lambdaImage) Password() string {
	return ""
}

func (s *lambdaImage) ResolveEnv(tokens ...string) (string, bool) {
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

// LambdaInvokedCount can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the count of times a named lambda function was invoked
//
//go:noinline
func LambdaInvokedCount(name string, imgName ...string) marrow.Resolvable {
	return &resolvable[LambdaService]{
		name:     fmt.Sprintf("LambdaInvokeCount(%q)", name),
		defImage: lambdaImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img LambdaService) (result any, err error) {
			return img.InvokedCount(name)
		},
		frame: framing.NewFrame(0),
	}
}
