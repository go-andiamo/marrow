package with

// Image is the interface that described a running docker image
type Image interface {
	Name() string
	Host() string
	Port() string
	MappedPort() string
	IsDocker() bool
	Username() string
	Password() string
}
