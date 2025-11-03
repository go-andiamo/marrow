package schema

import (
	"embed"
)

//go:embed *.sql
var Migrations embed.FS
