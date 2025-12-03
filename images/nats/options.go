package nats

import "time"

type Options struct {
	ImageVersion          string // defaults to "latest"
	Image                 string // defaults to "nats"
	Username              string // defaults to "nats"
	Password              string // defaults to "nats"
	SecretToken           string // defaults to "nats"
	LeaveRunning          bool   // if set, the container is not shutdown
	CreateKeyValueBuckets map[string]KeyValueBucket
}

type KeyValueBucket struct {
	Description  string
	History      uint8
	TTL          time.Duration
	MaxValueSize int32
	MaxBytes     int64
	Storage      int
	Compression  bool
}

const (
	defaultImage    = "nats"
	defaultVersion  = "latest"
	defaultUser     = "nats"
	defaultPassword = "nats"
	defaultSecret   = "nats"
)

func (o Options) image() string {
	if o.Image != "" {
		return o.Image
	}
	return defaultImage
}

func (o Options) version() string {
	if o.ImageVersion != "" {
		return o.ImageVersion
	}
	return defaultVersion
}

func (o Options) useImage() string {
	return o.image() + ":" + o.version()
}

func (o Options) username() string {
	if o.Username != "" {
		return o.Username
	}
	return defaultUser
}

func (o Options) password() string {
	if o.Password != "" {
		return o.Password
	}
	return defaultPassword
}

func (o Options) secret() string {
	if o.SecretToken != "" {
		return o.SecretToken
	}
	return defaultSecret
}
