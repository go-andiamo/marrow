package main

import (
	"appaws/api"
	"appaws/cloud"
	"appaws/config"
	"appaws/repository"
	"context"
	"fmt"
)

func main() {
	acs, err := cloud.NewAwsClients(context.Background(), "/app/petstore")
	if err != nil {
		panic(err)
	}
	cfg, err := config.Load(context.Background(), acs)
	if err != nil {
		panic(fmt.Errorf("loading config: %w", err))
	}

	fmt.Printf("config: %+v\n", cfg)

	repo, err := repository.NewRepository(*cfg)
	if err != nil {
		panic(fmt.Errorf("opening db: %w", err))
	}
	a := api.NewApi(*cfg, repo, acs)
	a.Start()
}
