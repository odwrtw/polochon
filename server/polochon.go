package main

import (
	"log"

	// Modules
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/eztv"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/fsnotify"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/openguessit"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/tmdb"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/tvdb"
	_ "gitlab.quimbo.fr/odwrtw/polochon/modules/yts"
)

func main() {
	configPath := "../config.yml"

	app, err := NewApp(configPath)
	if err != nil {
		log.Panic(err)
	}

	// pretty.Println(app)

	app.Run()

	// err = app.Organize()
	// if err != nil {
	// 	log.Panic(err)
	// }

	// // ------------------------------
	// //		  Guess
	// // ------------------------------
	// filePath := "/home/greg/downloads/done/American Dad! - 12x03 - Scents and Sensei-bility.mp4"
	// file := polochon.NewFile(filePath)
	// v, err := file.Guess(config.Video.Guesser)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// err = v.GetDetails(config.Show.Detailer)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// err = v.Store(config)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// pretty.Println(v)

	// ------------------------------
	//		   Movie store
	// ------------------------------
	// vs := polochon.NewVideoStore(config)

	// if err := vs.Scan(); err != nil {
	// 	log.Panic(err)
	// }
	// pretty.Println(vs)

	// pretty.Println(config)

	// ------------------------------
	//			   Movie
	// ------------------------------
	// m := polochon.NewMovie()
	// m.Title = "matrix"
	// // m.ImdbID = "tt0133093"

	// err = m.GetDetails(config.Movie.Detailers)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// err = m.GetTorrents(config.Movie.Torrenters)
	// if err != nil {
	// 	log.Panic(err)
	// }
	// pretty.Println(m)

	// m.File = polochon.NewFile("/home/greg/movies/The Matrix (1999)/matrix.mp4")

	// err = m.Store(config)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// ------------------------------
	//			   Show
	// ------------------------------
	// s := polochon.NewShow()
	// s.ImdbID = "tt0397306"
	// s.Title = "American dad"

	// s := polochon.NewShowEpisode()
	// s.Episode = 3
	// s.Season = 5
	// s.ShowTitle = "Game of thrones"

	// err = s.GetDetails(config.Show.Detailer)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// err = s.Store(config)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// err = s.GetTorrents(config.Show.Torrenter)
	// if err != nil {
	// 	log.Panic(err)
	// }

	// pretty.Println(s)
}
