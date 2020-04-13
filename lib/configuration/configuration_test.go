package configuration

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/modules/mock"
	"github.com/robfig/cron/v3"
)

var testConfigData = []byte(`
logs:
  level: panic
watcher:
  fsnotifier: mock
  dir: /downloads/todo
downloader:
  enabled: true
  launch_at_startup: true
  schedule: "@every 4h"
  client: mock
download_manager:
  enabled: true
  timer: 30s
  ratio: 0
  dir: /downloads
http_server:
  enable: true
  port: 8080
  host: localhost
  serve_files: true
  basic_auth: false
  basic_auth_user: toto
  basic_auth_password: tata
wishlist:
  wishlisters:
  - mock
  show_default_qualities:
  - 720p
  - 480p
  - 1080p
  movie_default_qualities:
  - 1080p
  - 720p
video:
  guesser: mock
  notifiers:
  - mock
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
  - fr_FR
  - en_US
show:
  calendar: mock
  dir: /tmp
  torrenters:
    - mock
  detailers:
    - mock
  subtitlers:
    - mock
movie:
  dir: /tmp
  torrenters:
    - mock
  detailers:
    - mock
  subtitlers:
    - mock
modules_params:
  - name: mock
`)

func TestReadConfig(t *testing.T) {
	polochon.ClearRegisteredModules()
	mock := &mock.Mock{}
	polochon.RegisterModule(mock)

	buf := bytes.NewBuffer(testConfigData)
	got, err := LoadConfig(buf)
	if err != nil {
		t.Fatalf("should not get any error but got %q", err)
	}

	// We don't really care about the logger, let's only check that it's not nil
	if got.Logger == nil {
		t.Fatalf("got a nil logger")
	}
	got.Logger = nil

	expected := &Config{
		Watcher: WatcherConfig{
			Dir:        "/downloads/todo",
			FsNotifier: mock,
		},
		Downloader: DownloaderConfig{
			Enabled:         true,
			LaunchAtStartup: true,
			Schedule: cron.ConstantDelaySchedule{
				Delay: 4 * time.Hour,
			},
			Client: mock,
		},
		DownloadManager: DownloadManagerConfig{
			Enabled: true,
			Dir:     "/downloads",
			Timer:   30 * time.Second,
			Ratio:   0,
		},
		HTTPServer: HTTPServer{
			Enable:            true,
			Port:              8080,
			Host:              "localhost",
			ServeFiles:        true,
			BasicAuth:         false,
			BasicAuthUser:     "toto",
			BasicAuthPassword: "tata",
		},
		Wishlist: polochon.WishlistConfig{
			Wishlisters:           []polochon.Wishlister{mock},
			ShowDefaultQualities:  []polochon.Quality{"720p", "480p", "1080p"},
			MovieDefaultQualities: []polochon.Quality{"1080p", "720p"},
		},
		Movie: polochon.MovieConfig{
			Torrenters: []polochon.Torrenter{mock},
			Detailers:  []polochon.Detailer{mock},
			Subtitlers: []polochon.Subtitler{mock},
		},
		Show: polochon.ShowConfig{
			Calendar:   mock,
			Torrenters: []polochon.Torrenter{mock},
			Detailers:  []polochon.Detailer{mock},
			Subtitlers: []polochon.Subtitler{mock},
		},
		File: polochon.FileConfig{
			ExcludeFileContaining:     []string{"sample"},
			VideoExtensions:           []string{".avi", ".mkv", ".mp4"},
			AllowedExtensionsToDelete: []string{".srt", ".nfo", ".txt", ".jpg", ".jpeg"},
			Guesser:                   mock,
		},
		Library: LibraryConfig{
			MovieDir: "/tmp",
			ShowDir:  "/tmp",
		},
		Notifiers:         []polochon.Notifier{mock},
		SubtitleLanguages: []polochon.Language{"fr_FR", "en_US"},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("invalid configuration\ngot:\n%+v\nexpected:\n%+v", got, expected)
	}
}
