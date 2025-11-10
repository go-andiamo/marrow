package with

// Image is the interface that describes a running docker image
type Image interface {
	Name() string
	Host() string
	Port() string
	MappedPort() string
	IsDocker() bool
	Username() string
	Password() string
}

// ImageResolveEnv is an additional interface that images can implement to resolve additional env settings
type ImageResolveEnv interface {
	ResolveEnv(tokens ...string) (string, bool)
}
