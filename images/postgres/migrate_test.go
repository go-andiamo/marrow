package postgres

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_migrateDatabase_NoMigrations(t *testing.T) {
	img := &image{}
	err := img.migrateDatabase()
	require.NoError(t, err)
}

func Test_migrateDatabase_ErrorsWithNoDbName(t *testing.T) {
	img := &image{
		options: Options{
			Migrations: []Migration{
				{},
			},
		},
	}
	err := img.migrateDatabase()
	require.Error(t, err)
}
