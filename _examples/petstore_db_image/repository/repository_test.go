package repository

import (
	"app/repository/schema"
	"app/repository/schema/seeds"
	"context"
	"database/sql"
	"github.com/go-andiamo/marrow/images/mysql"
	"os"
	"strings"
	"testing"
)

var testRepo Repository
var testDb *sql.DB

func TestMain(m *testing.M) {
	// demonstrates that db images can also be used in unit tests...
	img := mysql.With("", mysql.Options{
		Database: "petstore",
		Migrations: []mysql.Migration{
			{
				Filesystem: schema.Migrations,
			},
			{
				Filesystem: seeds.Migrations,
				TableName:  "schema_migrations_seeds",
			},
		},
	})
	err := img.Start()
	if err != nil {
		panic(err)
	}
	testDb = img.Database()
	testRepo = &repository{
		db: testDb,
	}

	ec := m.Run()
	// shutdown image - rather than waiting for reaper...
	img.Shutdown()()

	os.Exit(ec)
}

type rawQuery string

func insertDb(t *testing.T, tableName string, row map[string]any) {
	cols := make([]string, 0, len(row))
	args := make([]any, 0, len(row))
	markers := make([]string, 0, len(row))
	for k, v := range row {
		cols = append(cols, k)
		switch vt := v.(type) {
		case rawQuery:
			markers = append(markers, string(vt))
		default:
			args = append(args, v)
			markers = append(markers, "?")
		}
	}
	query := `INSERT INTO ` + tableName + ` (` + strings.Join(cols, ",") + `) VALUES (` + strings.Join(markers, ",") + `)`
	_, err := testDb.ExecContext(context.Background(), query, args...)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCategoriesSeeded(t *testing.T) {
	count := 0
	err := testDb.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count)
	if err != nil {
		t.Error(err)
	}
	if count < 3 {
		t.Error("Expected at least 3 categories")
	}
}
