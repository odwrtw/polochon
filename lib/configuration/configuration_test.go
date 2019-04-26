package configuration

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/modules/mock"
)

var testConfigData = []byte(`
logs:
  level: panic
watcher:
  fsnotifier: mock
  dir: /tmp
downloader:
  enabled: false
  timer: 4h
  client: mock
  cleaner:
    enabled: true
    timer: 30s
    ratio: 0
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

	mock := &mock.Mock{}
	expected := &Config{
		Watcher: WatcherConfig{
			Dir:        "/tmp",
			FsNotifier: mock,
		},
		Downloader: DownloaderConfig{
			Enabled: false,
			Timer:   4 * 3600 * time.Second,
			Client:  mock,
			Cleaner: CleanerConfig{

				Enabled: true,
				Timer:   30 * time.Second,
				Ratio:   0,
			},
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
			VideoExtentions:           []string{".avi", ".mkv", ".mp4"},
			AllowedExtentionsToDelete: []string{".srt", ".nfo", ".txt", ".jpg", ".jpeg"},
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
