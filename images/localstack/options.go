package localstack

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/go-andiamo/marrow/with"
	"time"
)

type Options struct {
	ImageVersion        string // defaults to "latest"
	Image               string // defaults to "localstack/localstack"
	LeaveRunning        bool   // if set, the container is not shutdown
	Region              string // defaults to "us-east-1"
	AccessKey           string // defaults to "dummy"
	SecretKey           string // defaults to "dummy"
	SessionToken        string // defaults to "SESSION"
	Services            Services
	Dynamo              DynamoOptions
	S3                  S3Options
	SNS                 SNSOptions
	SQS                 SQSOptions
	SecretsManager      SecretsManagerOptions
	Lambda              LambdaOptions
	CustomServices      CustomServiceBuilders
	DisableAutoShutdown bool // Deprecated: use with.DisableReaperShutdowns instead
}

type CustomServiceBuilders = []func(ctx context.Context, awsCfg aws.Config, host string, mappedPort string) (image with.Image, err error)

type DynamoOptions struct {
	CreateTables []dynamodb.CreateTableInput
}

type S3Options struct {
	CreateBuckets []s3.CreateBucketInput
}

type SNSOptions struct {
	CreateTopics []sns.CreateTopicInput
	// TopicsSubscribe if set to true, will subscribe to the created topics
	//
	// subscribing means that messages on those topics will be captured and made
	// available during tests
	TopicsSubscribe bool
	// MaxMessages is the maximum number of messages to store (it does not limit the counts)
	MaxMessages int
	// Unmarshaler is an optional message unmarshaler
	Unmarshaler func(msg SnsMessage) any
	// JsonMessages if set, treats SnsMessage.Message as json
	JsonMessages bool
}

type SQSOptions struct {
	CreateQueues []sqs.CreateQueueInput
}

type SecretsManagerOptions struct {
	Secrets     map[string]string
	JsonSecrets map[string]any
}

type LambdaOptions struct {
	CreateFunctions []string
	ActiveTimeout   time.Duration // defaults to 1 minute per lambda
	PullTimeout     time.Duration // defaults to 5 minutes - to allow for runtime image pull
}

type Service int
type Services []Service

const (
	All            Service = iota // start all services
	Dynamo                        // start DynamoDB service
	S3                            // start S3 service
	SNS                           // start SNS service
	SQS                           // start SQS service
	SecretsManager                // start SecretsManager service
	Lambda                        // start Lambda service

	DynamoDB           = Dynamo
	maxService Service = Lambda + 1
	// Except services following this are not started, e.g.
	//    Options.Services = Services{All,Except,SQS}
	Except Service = -1
)

const (
	defaultVersion      = "latest"
	defaultImage        = "localstack/localstack"
	defaultRegion       = "us-east-1"
	defaultAccessKey    = "dummy"
	defaultSecretKey    = "dummy"
	defaultSessionToken = "SESSION"
)

func (o Options) version() string {
	if o.ImageVersion != "" {
		return o.ImageVersion
	}
	return defaultVersion
}

func (o Options) image() string {
	if o.Image != "" {
		return o.Image
	}
	return defaultImage
}

func (o Options) useImage() string {
	return o.image() + ":" + o.version()
}

func (o Options) region() string {
	if o.Region != "" {
		return o.Region
	}
	return defaultRegion
}

func (o Options) accessKey() string {
	if o.AccessKey != "" {
		return o.AccessKey
	}
	return defaultAccessKey
}

func (o Options) secretKey() string {
	if o.SecretKey != "" {
		return o.SecretKey
	}
	return defaultSecretKey
}

func (o Options) sessionToken() string {
	if o.SessionToken != "" {
		return o.SessionToken
	}
	return defaultSessionToken
}

func (o Options) services() map[Service]struct{} {
	result := make(map[Service]struct{}, len(o.Services))
	except := false
	all := Services{Dynamo, S3, SNS, SQS, SecretsManager, Lambda}
	for _, service := range o.Services {
		switch service {
		case All:
			if !except {
				for _, s := range all {
					result[s] = struct{}{}
				}
			}
		case Except:
			except = true
		case Dynamo, S3, SNS, SQS, SecretsManager, Lambda:
			if !except {
				result[service] = struct{}{}
			} else {
				delete(result, service)
			}
		default:
			if !except && service < 0 {
				minusService := -service
				switch minusService {
				case Dynamo, S3, SNS, SQS, SecretsManager, Lambda:
					delete(result, minusService)
				}
			}
		}
	}
	return result
}
