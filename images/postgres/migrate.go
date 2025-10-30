package postgres

import (
	"errors"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

func (i *image) migrateDatabase() (err error) {
	if len(i.options.Migrations) == 0 {
		return nil
	} else if i.options.Database == "" {
		return errors.New("database name is required to migrate")
	}
	defer func() {
		if err != nil {
			err = fmt.Errorf("migrate database: %w", err)
		}
	}()
	for m := 0; m < len(i.options.Migrations) && err == nil; m++ {
		if migration := i.options.Migrations[m]; migration.Filesystem != nil {
			var driver database.Driver
			if driver, err = postgres.WithInstance(i.db, &postgres.Config{DatabaseName: i.options.Database, MigrationsTable: migration.TableName}); err == nil {
				var sourceDriver source.Driver
				if sourceDriver, err = iofs.New(migration.Filesystem, migration.path()); err == nil {
					err = migrateWithSource(sourceDriver, driver, i.options.Database)
				}
			}
		}
	}
	return err
}

func migrateWithSource(sourceDriver source.Driver, dbDriver database.Driver, dbName string) (err error) {
	defer func() {
		_ = sourceDriver.Close()
	}()
	var migrator *migrate.Migrate
	if migrator, err = migrate.NewWithInstance("iofs", sourceDriver, dbName, dbDriver); err == nil {
		err = migrator.Up()
	}
	return err
}
