package mysql

import (
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
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
			if driver, err = mysql.WithInstance(i.db, &mysql.Config{MigrationsTable: migration.TableName}); err == nil {
				var sourceDriver source.Driver
				if sourceDriver, err = iofs.New(migration.Filesystem, migration.path()); err == nil {
					err = migrateWithSource(sourceDriver, driver, i.options.Database, "")
				}
			}
		}
	}
	return err
}

func migrateWithSource(sourceDriver source.Driver, dbDriver database.Driver, dbName string, seed string) error {
	if migrator, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, dbDriver); err == nil {
		err = migrator.Up()
	}
	return nil
}
