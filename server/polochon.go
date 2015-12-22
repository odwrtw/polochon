package main

import (
	"flag"
	"log"
	"os"

	"github.com/odwrtw/polochon/token"
)

func fileExist(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func main() {
	configPath := flag.String("configPath", "../config.yml", "path of the configuration file")
	tokenPath := flag.String("tokenPath", "../token.yml", "path of the token file")
	flag.Parse()

	var tokenManager *token.Manager

	if fileExist(*tokenPath) {
		var err error

		file, err := os.Open(*tokenPath)
		defer file.Close()
		if err != nil {
			log.Panic(err)
		}

		tokenManager, err = token.LoadFromYaml(file)
		if err != nil {
			log.Panic(err)
		}
	}

	app, err := NewApp(*configPath, tokenManager)
	if err != nil {
		log.Panic(err)
	}

	app.Run()
}
