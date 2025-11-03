package postgres

import "io/fs"

type Options struct {
	ImageVersion string // defaults to "16.10-alpine3.21"
	Image        string // defaults to "postgres"
	RootUser     string // defaults to "root"
	RootPassword string // defaults to "root"
	// Database is the database (schema) name to use
	// If this is a non-empty string, the database will be created
	Database            string
	DefaultPort         string // is the actual port for Postgres, defaults to "5432"
	DisableAutoShutdown bool   // if set, disables container auto (RYUK reaper) shutdown
	LeaveRunning        bool   // if set, the container is not shutdown
	Migrations          []Migration
}

type Migration struct {
	Filesystem fs.FS
	Path       string // defaults to "." (all files in the supplied Filesystem)
	TableName  string // is the migration table name for this migration (defaults to "schema_migrations")
}

func (m Migration) path() string {
	if m.Path != "" {
		return m.Path
	}
	return "."
}

const (
	defaultVersion  = "16.10-alpine3.21"
	defaultImage    = "postgres"
	defaultUsername = "root"
	defaultPassword = "root"
	defaultPort     = "5432"
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
