package configuration

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

var configFileExample = []byte(`
watcher:
  fsnotifier: fsnotify
  dir: /home/user/downloads
downloader:
  timer: 30s
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

// Fake is a structure which implements a shitload of stuff
type Fake struct{}

func (f *Fake) Name() string {
	return "fake"
}

func (f *Fake) Init([]byte) error {
	return nil
}

func (f *Fake) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusOK, nil
}

func (f *Fake) GetTorrents(interface{}, *logrus.Entry) error {
	return nil
}

func (f *Fake) SearchTorrents(string) ([]*polochon.Torrent, error) {
	return nil, nil
}

func (f *Fake) GetDetails(interface{}, *logrus.Entry) error {
	return nil
}

func (f *Fake) GetSubtitle(interface{}, polochon.Language, *logrus.Entry) (polochon.Subtitle, error) {
	return nil, nil
}

func (f *Fake) AvailableMovieOptions() []string {
	return nil
}

func (f *Fake) GetMovieList(option string, log *logrus.Entry) ([]*polochon.Movie, error) {
	return nil, nil
}

func (f *Fake) AvailableShowOptions() []string {
	return nil
}

func (f *Fake) GetShowList(option string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, nil
}

func (f *Fake) SearchMovie(key string, log *logrus.Entry) ([]*polochon.Movie, error) {
	return nil, nil
}

func (f *Fake) SearchShow(key string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, nil
}

func TestModulesStatus(t *testing.T) {
	fake := &Fake{}
	c := Config{
		Movie: polochon.MovieConfig{
			Torrenters: []polochon.Torrenter{fake},
			Detailers:  []polochon.Detailer{fake},
			Subtitlers: []polochon.Subtitler{fake},
			Explorers:  []polochon.Explorer{fake},
			Searchers:  []polochon.Searcher{fake},
		},
		Show: polochon.ShowConfig{
			Torrenters: []polochon.Torrenter{fake},
		},
	}
	modulesStatus := c.ModulesStatus()
	expectedModulesStatus := ModulesStatuses{
		"movie": {
			"searcher": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"detailer": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"explorer": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"torrenter": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
			"subtitler": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
		},
		"show": {
			"torrenter": []ModuleStatus{
				{
					Name:   "fake",
					Status: polochon.StatusOK,
					Error:  "",
				},
			},
		},
	}
	if !reflect.DeepEqual(modulesStatus, expectedModulesStatus) {
		t.Errorf("Didn't get expected module status \n %+v \n %+v", modulesStatus, expectedModulesStatus)
	}
}
