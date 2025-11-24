package main

import (
	"app/api"
	"app/config"
	"app/repository"
	. "github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/common"
	"github.com/go-andiamo/marrow/images/mysql"
	"github.com/go-andiamo/marrow/with"
	"testing"
)

// Run the same Suite but with the API run from code rather than docker container
// enabling the ability to debug
func TestApi(t *testing.T) {
	// spin up supporting db as docker container...
	img := mysql.With("", dbOptions)
	if err := img.Start(); err != nil {
		t.Fatal(err)
	}
	// create the repo on existing db (from container)...
	repo, _ := repository.NewRepositoryWithDB(img.Database())

	// create and start the api locally...
	a := api.NewApi(config.Api{Port: 8080}, repo)
	go a.Start()

	// now run the suite...
	err := Suite(endpoints...).Init(
		with.Var(string(nonExistentId), "00000000-0000-485c-0000-000000000000"),
		with.Database("", img.Database(), common.DatabaseArgs{Style: common.PositionalDbArgs}),
		with.ApiHost("localhost", 8080),
		with.Testing(t),
	).Run()
	if err != nil {
		t.Fatal(err)
	}
}
