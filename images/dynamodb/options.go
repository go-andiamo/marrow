package dynamodb

import "github.com/aws/aws-sdk-go-v2/service/dynamodb"

type Options struct {
	ImageVersion        string // defaults to "2.1.0"
	Image               string // defaults to "localstack/localstack"
	DefaultPort         string // is the actual port for dynamodb, defaults to "4566"
	DisableAutoShutdown bool   // if set, disables container auto (RYUK reaper) shutdown
	LeaveRunning        bool   // if set, the container is not shutdown
	Region              string // defaults to "us-east-1"
	CreateTables        []dynamodb.CreateTableInput
}

const (
	defaultVersion = "2.1.0"
	defaultImage   = "localstack/localstack"
	defaultPort    = "4566"
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

func (o Options) defaultPort() string {
	if o.DefaultPort != "" {
		return o.DefaultPort
	}
	return defaultPort
}
