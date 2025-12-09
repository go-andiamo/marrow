package config

import (
	"appaws/cloud"
	"context"
	"errors"
	"github.com/go-andiamo/cfgenv"
	"os"
)

func Load(ctx context.Context, aws *cloud.AwsClients) (*Config, error) {
	if aws == nil {
		if os.Getenv("USE_LOCAL_ENV") == "true" {
			return LoadFromEnv()
		}
		return nil, errors.New("aws clients is nil")
	}
	cfg := &Config{}
	var err error
	if cfg.Api.Port, err = aws.GetParameter(ctx, "api_port"); err != nil {
		return nil, err
	}
	if cfg.Database.Host, err = aws.GetParameter(ctx, "db_host"); err != nil {
		return nil, err
	}
	if cfg.Database.Port, err = aws.GetParameter(ctx, "db_port"); err != nil {
		return nil, err
	}
	if cfg.Database.Name, err = aws.GetParameter(ctx, "db_name"); err != nil {
		return nil, err
	}
	if cfg.Database.Username, err = aws.GetSecret(ctx, "db_username"); err != nil {
		return nil, err
	}
	if cfg.Database.Password, err = aws.GetSecret(ctx, "db_password"); err != nil {
		return nil, err
	}
	if cfg.Topics.Pets, err = aws.GetParameter(ctx, "pets_topic_arn"); err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadFromEnv() (*Config, error) {
	return cfgenv.LoadAs[Config]()
}

type Config struct {
	Api      Api       `env:"prefix=API" json:"api"`
	Database Database  `env:"prefix=DATABASE" json:"database"`
	Topics   TopicArns `env:"prefix=TOPICS" json:"topics"`
}

type Api struct {
	Port string `json:"port"`
}

type Database struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type TopicArns struct {
	Pets string `json:"pets"`
}
