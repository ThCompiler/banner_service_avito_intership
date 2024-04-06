package config

import (
	"fmt"

	"bannersrv/pkg/logger"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		Port       string     `yaml:"port"`
		Postgres   PG         `yaml:"postgres"`
		Redis      Redis      `yaml:"redis"`
		LoggerInfo LoggerInfo `yaml:"logger"`
	}

	LoggerInfo struct {
		AppName           string          `yaml:"app_name"`
		Directory         string          `yaml:"directory"`
		Level             logger.LogLevel `yaml:"level"`
		UseStdAndFile     bool            `yaml:"use_std_and_file"`
		AllowShowLowLevel bool            `yaml:"allow_show_low_level"`
	}

	PG struct {
		URL string `yaml:"url"`
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
