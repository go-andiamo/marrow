package main

import (
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/with"
	"petstore/api"
	"testing"
)

func TestApi(t *testing.T) {
	port := 8080

	// start the api we're testing...
	a := api.NewApi()
	go a.Start(port)

	// initialise the suite...
	s := marrow.Suite(endpoints...).Init(
		with.ApiHost("localhost", port),
		with.Testing(t),
		with.Var(string(nonExistentId), "00000000-0000-485c-0000-000000000000"),
	)

	// run the suite...
	err := s.Run()
	if err != nil {
		t.Fatal(err)
	}
}
