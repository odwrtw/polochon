package main

import (
	"flag"
	"log"
)

func main() {
	configPath := flag.String("configPath", "../config.yml", "path of the configuration file")
	flag.Parse()

	app, err := NewApp(*configPath)
	if err != nil {
		log.Panic(err)
	}

	app.Run()
}
