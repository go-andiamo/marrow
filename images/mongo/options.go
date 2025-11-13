package mongo

import "go.mongodb.org/mongo-driver/v2/mongo"

type Options struct {
	ImageVersion        string // defaults to "7"
	Image               string // defaults to "mongo"
	RootUser            string // defaults to "root"
	RootPassword        string // defaults to "root"
	DefaultPort         string // is the actual port for Mongo, defaults to "27017"
	DisableAutoShutdown bool   // if set, disables container auto (RYUK reaper) shutdown
	LeaveRunning        bool   // if set, the container is not shutdown
	CreateIndices       IndexOptions
}

type IndexOptions map[string]map[string][]mongo.IndexModel

//                    ^ db       ^ coll

const (
	defaultVersion  = "7"
	defaultImage    = "mongo"
	defaultUsername = "root"
	defaultPassword = "root"
	defaultPort     = "27017"
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

func (o Options) username() string {
	if o.RootUser != "" {
		return o.RootUser
	}
	return defaultUsername
}

func (o Options) password() string {
	if o.RootPassword != "" {
		return o.RootPassword
	}
	return defaultPassword
}

func (o Options) defaultPort() string {
	if o.DefaultPort != "" {
		return o.DefaultPort
	}
	return defaultPort
}
