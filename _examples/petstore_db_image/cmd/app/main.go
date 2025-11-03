package main

import (
	"app/api"
	"app/config"
	"app/repository"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	repo, err := repository.NewRepository(cfg.Database)
	if err != nil {
		panic(err)
	}

	a := api.NewApi(cfg.Api, repo)
	a.Start()
}
