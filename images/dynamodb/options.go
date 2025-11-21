package dynamodb

import "github.com/aws/aws-sdk-go-v2/service/dynamodb"

type Options struct {
	ImageVersion string // defaults to "latest"
	Image        string // defaults to "localstack/localstack"
	DefaultPort  string // is the actual port for dynamodb, defaults to "4566"
	LeaveRunning bool   // if set, the container is not shutdown
	Region       string // defaults to "us-east-1"
	// CreateTables is a list of tables to be created in DynamoDB
	CreateTables        []dynamodb.CreateTableInput
	DisableAutoShutdown bool // Deprecated: use with.DisableReaperShutdowns instead
}

const (
	defaultVersion = "latest"
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
