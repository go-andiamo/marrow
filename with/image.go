package with

type Image interface {
	Name() string
	Host() string
	Port() string
	MappedPort() string
	IsDocker() bool
	Username() string
	Password() string
}
