package main

import (
	"flag"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/app/internal/app"
)

func main() {
	configPath := flag.String("configPath", "../config.yml", "path of the configuration file")
	tokenPath := flag.String("tokenPath", "", "path of the token file")
	flag.Parse()

	app, err := app.NewApp(*configPath, *tokenPath)
	if err != nil {
		logrus.Fatal(err)
	}

	app.Run()
}
