package configuration

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/odwrtw/polochon/lib"
)

var configFileExample = []byte(`
watcher:
  fsnotifier: fsnotify
  dir: /home/user/downloads
downloader:
  timer: 30s
  download_dir: /home/user/downloads
  client: transmission
http_server:
  port: 8080
  host: localhost
  enable: true
  serve_files: true
  basic_auth: true
  basic_auth_user: toto
  basic_auth_password: tata
wishlist:
  wishlisters:
  - imdb
  - canape
  show_default_qualities:
  - 720p
  - 480p
  - 1080p
  movie_default_qualities:
  - 1080p
  - 720p
video:
  guesser: openguessit
  notifiers:
  - pushover
  exclude_file_containing:
  - sample
  allowed_file_extensions:
  - .avi
  - .mkv
  - .mp4
  allowed_file_extensions_to_delete:
  - .srt
  - .nfo
  - .txt
  - .jpg
  - .jpeg
  subtitle_languages:
  - en_US
  - fr_FR
modules_params:
  - name: pushover
    user: 9327a472s3947234792
    key: sdf7as8f8ds7f9sf
  - name: addicted
    lang: fr_FR
  - name: opensubtitles
    lang: en_US
    user: myUserName
    password: myPass
show:
  calendar: tvdb
  dir: /home/user/tvshows
  torrenters:
    - eztv
  detailers:
    - tvdb
  subtitlers:
    - addicted
movie:
  dir: /home/user/movies
  torrenters:
    - yts
  detailers:
    - tmdb
  subtitlers:
    - opensubtitles
`)

var configStructExample = &ConfigFileRoot{
	Watcher: ConfigFileWatcher{
		Dir:            "/home/user/downloads",
		FsNotifierName: "fsnotify",
	},
	Downloader: ConfigFileDownloader{
		Timer:          time.Second * 30,
		DownloadDir:    "/home/user/downloads",
		DownloaderName: "transmission",
	},
	HTTPServer: ConfigFileHTTPServer{
		Enable:            true,
		Port:              8080,
		Host:              "localhost",
		ServeFiles:        true,
		BasicAuth:         true,
		BasicAuthUser:     "toto",
		BasicAuthPassword: "tata",
	},
	Wishlist: ConfigFileWishlist{
		WishlisterNames:       []string{"imdb", "canape"},
		ShowDefaultQualities:  []polochon.Quality{polochon.Quality720p, polochon.Quality480p, polochon.Quality1080p},
		MovieDefaultQualities: []polochon.Quality{polochon.Quality1080p, polochon.Quality720p},
	},
	Video: ConfigFileVideo{
		ExcludeFileContaining:     []string{"sample"},
		VideoExtentions:           []string{".avi", ".mkv", ".mp4"},
		AllowedExtentionsToDelete: []string{".srt", ".nfo", ".txt", ".jpg", ".jpeg"},
		NotifierNames:             []string{"pushover"},
		GuesserName:               "openguessit",
		SubtitleLanguages:         []polochon.Language{polochon.EN, polochon.FR},
	},
	ModulesParams: []map[string]interface{}{
		{
			"name": "pushover",
			"user": "9327a472s3947234792",
			"key":  "sdf7as8f8ds7f9sf",
		},
		{
			"name": "addicted",
			"lang": "fr_FR",
		},
		{
			"name":     "opensubtitles",
			"lang":     "en_US",
			"user":     "myUserName",
			"password": "myPass",
		},
	},
	Movie: ConfigFileMovie{
		Dir:            "/home/user/movies",
		TorrenterNames: []string{"yts"},
		DetailerNames:  []string{"tmdb"},
		SubtitlerNames: []string{"opensubtitles"},
	},
	Show: ConfigFileShow{
		Dir:            "/home/user/tvshows",
		TorrenterNames: []string{"eztv"},
		DetailerNames:  []string{"tvdb"},
		SubtitlerNames: []string{"addicted"},
		CalendarName:   "tvdb",
	},
}

func TestReadConfig(t *testing.T) {
	b := bytes.NewBuffer(configFileExample)

	got, err := readConfig(b)
	if err != nil {
		t.Fatalf("failed to read config from file: %q", err)
	}

	if !reflect.DeepEqual(got, configStructExample) {
		t.Errorf("Didn't get expected config \n %+v \n %+v", got, configStructExample)
	}
}
