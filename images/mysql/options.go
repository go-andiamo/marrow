package mysql

import "io/fs"

type Options struct {
	ImageVersion string // defaults to "8.0.40"
	Image        string // defaults to "docker.io/mysql"
	RootUsername string // defaults to "root"
	RootPassword string // defaults to "root"
	// Database is the database (schema) name to use
	// If this is a non-empty string, the database will be created
	Database            string
	DefaultPort         string      // is the actual port for MySql, defaults to "3306"
	LeaveRunning        bool        // if set, the container is not shutdown
	Migrations          []Migration // is a list of Migration's to be run on the database
	DisableAutoShutdown bool        // Deprecated: use with.DisableReaperShutdowns instead
}

// Migration is an individual migration to be run in a database
type Migration struct {
	Filesystem fs.FS  // the file system containing the migration .up.sql and .down.sql files (see github.com/golang-migrate/migrate/v4)
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
	defaultVersion      = "8.0.40"
	defaultImage        = "docker.io/mysql"
	defaultRootUsername = "root"
	defaultRootPassword = "root"
	defaultPort         = "3306"
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

func (o Options) rootUsername() string {
	if o.RootUsername != "" {
		return o.RootUsername
	}
	return defaultRootUsername
}

func (o Options) rootPassword() string {
	if o.RootPassword != "" {
		return o.RootPassword
	}
	return defaultRootPassword
}

func (o Options) defaultPort() string {
	if o.DefaultPort != "" {
		return o.DefaultPort
	}
	return defaultPort
}
