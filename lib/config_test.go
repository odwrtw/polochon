package polochon

import (
	"bytes"
	"reflect"
	"testing"
	"time"
)

var configFileExample = []byte(`
watcher:
  timer: 30s
  fsnotifier: fsnotify
  dir: /home/user/downloads
downloader:
  download_dir: /home/user/downloads
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
		Timer:          time.Second * 30,
		Dir:            "/home/user/downloads",
		FsNotifierName: "fsnotify",
	},
	Downloader: ConfigFileDownloader{
		DownloadDir: "/home/user/downloads",
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
		WishlisterNames:       []string{"imdb"},
		ShowDefaultQualities:  []Quality{Quality720p, Quality480p, Quality1080p},
		MovieDefaultQualities: []Quality{Quality1080p, Quality720p},
	},
	Video: ConfigFileVideo{
		ExcludeFileContaining:     []string{"sample"},
		VideoExtentions:           []string{".avi", ".mkv", ".mp4"},
		AllowedExtentionsToDelete: []string{".srt", ".nfo", ".txt", ".jpg", ".jpeg"},
		NotifierNames:             []string{"pushover"},
		GuesserName:               "openguessit",
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
