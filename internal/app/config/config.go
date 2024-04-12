package config

import (
	"fmt"

	"bannersrv/pkg/logger"

	"github.com/ilyakaznacheev/cleanenv"
)

type Mode string

const (
	Release     Mode = "release"
	Debug       Mode = "debug"
	DebugProf   Mode = "debug+prof"
	ReleaseProf Mode = "release+prof"
)

type (
	Config struct {
		Port       string     `yaml:"port"`
		Postgres   PG         `yaml:"postgres"`
		Redis      Redis      `yaml:"redis"`
		LoggerInfo LoggerInfo `yaml:"logger"`
		Mode       Mode       `yaml:"mode"`
	}

	LoggerInfo struct {
		AppName           string          `yaml:"app_name"`
		Directory         string          `yaml:"directory"`
		Level             logger.LogLevel `yaml:"level"`
		UseStdAndFile     bool            `yaml:"use_std_and_file"`
		AllowShowLowLevel bool            `yaml:"allow_show_low_level"`
	}

	PG struct {
		URL                string `yaml:"url"`
		MaxConnections     int    `yaml:"max_connections" default:"5"`
		MinConnections     int    `yaml:"min_connections" default:"2"`
		TTLIDleConnections uint64 `yaml:"ttl_idle_connections" default:"10"`
	}

	Redis struct {
		URL string `yaml:"url"`
	}
)

func NewConfig(path string) (*Config, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig(path, cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %s", err)
	}

	return cfg, nil
}
