package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/odwrtw/polochon/app/app"
	"github.com/sirupsen/logrus"
)

var (
	// VersionString contains the latest described commit from build
	VersionString string
	// RevisionString contains the latest commit
	RevisionString string
)

func main() {
	configPath := flag.String("configPath", "../config.yml", "path of the configuration file")
	tokenPath := flag.String("tokenPath", "", "path of the token file")
	versionFlag := flag.Bool("version", false, "show version number and quit")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("polochon %s\nLatest commit: %s\n", VersionString, RevisionString[0:6])
		os.Exit(0)
	}

	app, err := app.NewApp(*configPath, *tokenPath)
	if err != nil {
		logrus.Fatal(err)
	}

	app.Run()
}
