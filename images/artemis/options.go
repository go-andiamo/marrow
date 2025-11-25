package artemis

import "github.com/go-stomp/stomp/v3"

type Options struct {
	ImageVersion string // defaults to "2.30.0-alpine"
	Image        string // defaults to "apache/activemq-artemis"
	Username     string // defaults to "artemis"
	Password     string // defaults to "artemis"
	LeaveRunning bool   // if set, the container is not shutdown
	// CreateQueues is a list of queues to create at startup
	CreateQueues []string
	// Subscribers is a map of the topic subscribers to setup - where the key is the topic name
	//
	// information from subscribers can be captured in tests
	Subscribers Receivers
	// Consumers is a map of the queue consumers to setup - where the key is the queue name
	//
	// information from consumers can be captured in tests
	Consumers Receivers
	// Marshaller is an optional func to marshal message body for publish/send
	Marshaller func(msg any) (body []byte, contentType string, err error)
}

type Receivers map[string]Receiver

type Receiver struct {
	// MaxMessages is the maximum number of messages to hold
	//
	// if this is zero, no messages are held - but still keeps count of messages received
	MaxMessages int
	// JsonMessages if set, will unmarshal messages to JSON (i.e. `map[string]any`)
	JsonMessages bool
	// Unmarshaler if provided, is used to unmarshal messages
	Unmarshaler func(msg *stomp.Message) any
}

const (
	defaultImage    = "apache/activemq-artemis"
	defaultVersion  = "2.30.0-alpine"
	defaultUser     = "artemis"
	defaultPassword = "artemis"
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
