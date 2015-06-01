package main

import (
	"flag"
	"log"

	// Modules
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/eztv"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/fsnotify"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/openguessit"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/pushover"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/tmdb"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/tvdb"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/yts"
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
