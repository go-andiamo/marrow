package main

import (
	"app/repository/schema"
	"app/repository/schema/seeds"
	. "github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/images/mysql"
	"github.com/go-andiamo/marrow/with"
	"os"
	"path/filepath"
)

var endpoints = []Endpoint_{
	Endpoint("/api", "Root",
		Method(GET, "Get root").
			AssertOK().
			AssertEqual(JsonPath(Body, "hello"), "world"),
		Endpoint("/pets", "Pets",
			Method(GET, "Get pets (empty)").
				AssertOK().
				AssertLen(Body, 0),
			Method(POST, "Create pet").
				RequestBody(JSON{
					"name": "Felix",
					"dob":  "2025-11-01",
					"category": JSON{
						"id": Query("", "SELECT id FROM categories"),
					},
				}).
				AssertCreated().
				SetVar(After, "created-pet-id", JsonPath(Body, "id")),
			Method(GET, "Get pets (non-empty)").
				AssertOK().
				AssertLen(Body, 1),
			Endpoint("/{petId}", "Pet",
				Method(GET, "Get pet (not found)").
					PathParam(Var("non-uuid")).
					AssertNotFound(),
				Method(GET, "Get pet").
					PathParam(Var("created-pet-id")).
					AssertOK(),
				Method(DELETE, "Delete pet (not found)").
					PathParam(Var("non-uuid")).
					AssertNotFound(),
				Method(DELETE, "Delete pet successful").
					SetVar(Before, "before-count", Query("", "SELECT COUNT(*) FROM pets")).
					PathParam(Var("created-pet-id")).
					AssertNoContent().
					AssertGreaterThan(Var("before-count"), Query("", "SELECT COUNT(*) FROM pets")),
			),
		),
		Endpoint("/categories", "Categories",
			SetVar(Before, "categoryId", Query("", "SELECT id FROM categories")),
			Method(GET, "Get categories").
				AssertOK().
				AssertGreaterThan(JsonPath(Body, LEN), 0),
			Endpoint("/{categoryId}", "Category",
				Method(GET, "Get category (not found)").
					PathParam(Var("non-uuid")).
					AssertNotFound(),
				Method(GET, "Get category (found)").
					PathParam(Var("categoryId")).
					AssertOK(),
			),
		),
	),
}

func main() {
	apiEnv := map[string]any{
		"API_PORT":          8080,
		"DATABASE_HOST":     "host.docker.internal",
		"DATABASE_PORT":     "{$mysql:mport}",
		"DATABASE_NAME":     "petstore",
		"DATABASE_USERNAME": "{$mysql:username}",
		"DATABASE_PASSWORD": "{$mysql:password}",
	}
	s := Suite(endpoints...)
	s = s.Init(
		with.Var("non-uuid", "00000000-0000-485c-0000-000000000000"),
		with.Make(with.Supporting, absPath("./Makefile"), 0, false),
		with.ApiImage("petstore", "latest", 8080, apiEnv, false),
		mysql.With("mysql", mysql.Options{
			Database: "petstore",
			//DisableAutoShutdown: true,
			//LeaveRunning:        true,
			Migrations: []mysql.Migration{
				{
					Filesystem: schema.Migrations,
				},
				{
					Filesystem: seeds.Migrations,
					TableName:  "schema_migrations_seeds",
				},
			},
		}))
	err := s.Run()
	if err != nil {
		panic(err)
	}
}

func absPath(path string) string {
	if !filepath.IsAbs(path) {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Join(cwd, path)
		}
	}
	return path
}
