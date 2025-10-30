package mysql

import (
	"embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

//go:embed _testdata/*.sql
var migrationFiles embed.FS

func TestImage_start(t *testing.T) {
	img := &image{
		options: Options{
			Database:            "foo",
			DisableAutoShutdown: true,
			Migrations: []Migration{
				{
					Filesystem: migrationFiles,
					Path:       "_testdata",
				},
			},
		},
	}
	err := img.Start()
	defer func() {
		img.shutdown()
	}()
	require.NoError(t, err)

	rows, err := img.db.Query("SHOW TABLES")
	require.NoError(t, err)
	defer rows.Close()
	names := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			require.NoError(t, err)
		}
		names[name] = true
	}
	assert.Len(t, names, 3)
	assert.True(t, names["people"])
	assert.True(t, names["addresses"])
	assert.True(t, names["schema_migrations"])
	//host.docker.internal
}
