package localstack

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Options struct {
	ImageVersion        string // defaults to "latest"
	Image               string // defaults to "localstack/localstack"
	DisableAutoShutdown bool   // if set, disables container auto (RYUK reaper) shutdown
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
}

type DynamoOptions struct {
	CreateTables []dynamodb.CreateTableInput
}

type S3Options struct {
	CreateBuckets []s3.CreateBucketInput
}

type SNSOptions struct {
	CreateTopics []sns.CreateTopicInput
}

type SQSOptions struct {
	CreateQueues []sqs.CreateQueueInput
}

type Service int
type Services []Service

const (
	Dynamo Service = iota
	S3
	SNS
	SQS
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
	for _, service := range o.Services {
		switch service {
		case Dynamo, S3, SNS, SQS:
			result[service] = struct{}{}
		}
	}
	return result
}
