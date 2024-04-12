package main

import (
	"bannersrv/internal/app"
	"bannersrv/internal/app/config"
	"flag"
	"log"
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
