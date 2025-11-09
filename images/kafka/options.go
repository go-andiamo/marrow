package kafka

import (
	"github.com/IBM/sarama"
	"time"
)

type Options struct {
	ImageVersion        string // defaults to "7.5.0"
	Image               string // defaults to "confluentinc/confluent-local"
	DefaultPort         string // is the actual port for kafka, defaults to "9093"
	ClusterId           string // defaults to "kraftCluster"
	GroupId             string // defaults to "test-group"
	DisableAutoShutdown bool   // if set, disables container auto (RYUK reaper) shutdown
	LeaveRunning        bool   // if set, the container is not shutdown
	// Subscribers is a map of the topic subscribers to setup - where the key is the topic name
	//
	// information from subscribers can be captured in tests
	Subscribers Subscribers
	// Wait is a delay duration used when starting - this is useful when Subscribers have been added and allows time
	// for the Kafka topics to be created (recommended value for this is 2-5 seconds)
	Wait time.Duration
	// InitialOffsetOldest if set, instructs consumer group to use initial offset oldest (otherwise, offset newest is used)
	InitialOffsetOldest bool
}

type Subscribers map[string]Subscriber

type Subscriber struct {
	// MaxMessages is the maximum number of messages to hold
	//
	// if this is zero, no messages are held - but still keeps count of messages received
	MaxMessages int
	// JsonMessages if set, will unmarshal messages to JSON (i.e. `map[string]any`)
	JsonMessages bool
	// Unmarshaler if provided, is used to unmarshal messages
	Unmarshaler func(msg Message) any
	// Mark is the mark to use when processing subscribed Kafka messages
	Mark string
}

const (
	defaultVersion   = "7.5.0"
	defaultImage     = "confluentinc/confluent-local"
	defaultPort      = "9093"
	defaultClusterId = "kraftCluster"
	defaultGroupId   = "test-group"
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

func (o Options) clusterId() string {
	if o.ClusterId != "" {
		return o.ClusterId
	}
	return defaultClusterId
}

func (o Options) groupId() string {
	if o.GroupId != "" {
		return o.GroupId
	}
	return defaultGroupId
}

func (o Options) offsetInitial() int64 {
	if o.InitialOffsetOldest {
		return sarama.OffsetOldest
	}
	return sarama.OffsetNewest
}
