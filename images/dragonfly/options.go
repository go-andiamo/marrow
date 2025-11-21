package dragonfly

type Options struct {
	ImageVersion string // defaults to "v1.34.2"
	Image        string // defaults to "ghcr.io/dragonflydb/dragonfly"
	DefaultPort  string // is the actual port for dragonfly, defaults to "6379"
	LeaveRunning bool   // if set, the container is not shutdown
	// Subscribers is a map of the topic subscribers to setup - where the key is the topic name
	//
	// information from subscribers can be captured in tests
	Subscribers Receivers
	// Consumers is a map of the queue consumers to setup - where the key is the queue name
	//
	// information from consumers can be captured in tests
	Consumers           Receivers
	DisableAutoShutdown bool // Deprecated: use with.DisableReaperShutdowns instead
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
	Unmarshaler func(msg string) any
}

const (
	defaultVersion = "v1.34.2"
	defaultImage   = "ghcr.io/dragonflydb/dragonfly"
	defaultPort    = "6379"
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
