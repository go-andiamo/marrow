package dragonfly

type Options struct {
	ImageVersion        string // defaults to "v1.34.2"
	Image               string // defaults to "ghcr.io/dragonflydb/dragonfly"
	DefaultPort         string // is the actual port for dragonfly, defaults to "6379"
	DisableAutoShutdown bool   // if set, disables container auto (RYUK reaper) shutdown
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
