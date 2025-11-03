package config

import (
	"github.com/go-andiamo/cfgenv"
)

func Load() (*Config, error) {
	return cfgenv.LoadAs[Config]()
}

type Config struct {
	Api      Api      `env:"prefix=API" json:"api"`
	Database Database `env:"prefix=DATABASE" json:"database"`
}

type Api struct {
	Port int `json:"port"`
}

type Database struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Password string `json:"-"`
}
