package main

import (
	"flag"
	"log"

	"bannersrv/internal/app/config"

	"bannersrv/internal/app"
)

func main() {
	var configPath string

	flag.StringVar(&configPath, "config", "./config/localhost-config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	app.Run(cfg)
}
