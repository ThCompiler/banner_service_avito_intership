package main

import (
	"bannersrv/internal/app"
	"flag"
)

func main() {
	var configPath string

	flag.StringVar(&configPath, "config", "./config/localhost-config.yaml", "path to config file")
	flag.Parse()

	app.Run(configPath)
}
