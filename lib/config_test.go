package polochon

import (
	"bytes"
	"io/ioutil"
	"os"
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
  serve_files_user: toto
  serve_files_pwd: tata
video:
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
show:
  dir: /home/user/tvshows
  torrenters:
    - eztv
  detailers:
    - tvdb
movie:
  dir: /home/user/movies
  torrenters:
    - yts
  detailers:
    - tmdb
`)

var configStructExample = &Config{
	Watcher: WatcherConfig{
		Timer:          time.Second * 30,
		Dir:            "/home/user/downloads",
		FsNotifierName: "fsnotify",
	},
	Downloader: DownloaderConfig{
		DownloadDir: "/home/user/downloads",
	},
	HTTPServer: HTTPServerConfig{
		Enable:         true,
		Port:           8080,
		Host:           "localhost",
		ServeFiles:     true,
		ServeFilesUser: "toto",
		ServeFilesPwd:  "tata",
	},
	Video: VideoConfig{
		ExcludeFileContaining:     []string{"sample"},
		VideoExtentions:           []string{".avi", ".mkv", ".mp4"},
		AllowedExtentionsToDelete: []string{".srt", ".nfo", ".txt", ".jpg", ".jpeg"},
		NotifierNames:             []string{"pushover"},
	},
	ModulesParams: []map[string]string{
		{
			"name": "pushover",
			"user": "9327a472s3947234792",
			"key":  "sdf7as8f8ds7f9sf",
		},
	},
	Movie: MovieConfig{
		Dir:            "/home/user/movies",
		TorrenterNames: []string{"yts"},
		DetailerNames:  []string{"tmdb"},
	},
	Show: ShowConfig{
		Dir:            "/home/user/tvshows",
		TorrenterNames: []string{"eztv"},
		DetailerNames:  []string{"tvdb"},
	},
}

func TestWriteReadConfig(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "polochon-config")
	if err != nil {
		t.Fatal("failed to create temp file", err)
	}
	defer os.Remove(tmpFile.Name())

	err = configStructExample.WriteConfigFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to write config file: %q", err)
	}

	got, err := ReadConfigFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to read config from file: %q", err)
	}

	if !reflect.DeepEqual(got, configStructExample) {
		t.Errorf("Invalid config")
	}
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
