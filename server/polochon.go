package main

import (
	"flag"
	"log"

	// Modules
	_ "github.com/odwrtw/polochon/modules/addicted"
	_ "github.com/odwrtw/polochon/modules/eztv"
	_ "github.com/odwrtw/polochon/modules/fsnotify"
	_ "github.com/odwrtw/polochon/modules/openguessit"
	_ "github.com/odwrtw/polochon/modules/opensubtitles"
	_ "github.com/odwrtw/polochon/modules/pushover"
	_ "github.com/odwrtw/polochon/modules/tmdb"
	_ "github.com/odwrtw/polochon/modules/tvdb"
	_ "github.com/odwrtw/polochon/modules/yts"
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
