package library

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	_ "github.com/odwrtw/polochon/modules/mock"
)

type mockLibrary struct {
	*Library
	httpServer *httptest.Server
	tmpDir     string
}

func (m *mockLibrary) cleanup() {
	// Remove the tmp directory
	if strings.HasPrefix(m.tmpDir, os.TempDir()) {
		os.RemoveAll(m.tmpDir)
	} else {
		panic("trying to work in a non temporary directory")
	}

	// Close the server
	if m.httpServer != nil {
		m.httpServer.Close()
	}
}

func newMockLibrary() (*mockLibrary, error) {
	// Get the mock detailer
	detailerModule, err := polochon.GetModule("mock", polochon.TypeDetailer)
	if err != nil {
		return nil, err
	}
	detailer := detailerModule.(polochon.Detailer)

	// Get the mock subtitler
	subtitlerModule, err := polochon.GetModule("mock", polochon.TypeSubtitler)
	if err != nil {
		return nil, err
	}
	subtitler := subtitlerModule.(polochon.Subtitler)

	// Create a temp dir
	tmpDir, err := ioutil.TempDir("", "polochon-library")
	if err != nil {
		return nil, err
	}

	// Create the library configuration
	config := configuration.LibraryConfig{
		MovieDir: filepath.Join(tmpDir, "movies"),
		ShowDir:  filepath.Join(tmpDir, "shows"),
	}

	// Create the folder to hold the movies, shows and downloads
	for _, path := range []string{
		config.MovieDir,
		config.ShowDir,
		filepath.Join(tmpDir, "downloads"),
	} {
		if err := os.Mkdir(path, os.ModePerm); err != nil {
			return nil, err
		}
	}

	// FileConfig
	fileConfig := polochon.FileConfig{
		VideoExtentions: []string{".mp4"},
	}

	// MovieConfig with the mock detailer
	movieConfig := polochon.MovieConfig{
		Detailers:  []polochon.Detailer{detailer},
		Subtitlers: []polochon.Subtitler{subtitler},
	}

	// ShowConfig with the mock detailer
	showConfig := polochon.ShowConfig{
		Detailers:  []polochon.Detailer{detailer},
		Subtitlers: []polochon.Subtitler{subtitler},
	}

	// downloaderConfig is enabled
	downloaderConfig := configuration.DownloaderConfig{
		Enabled: true,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "mockContent")
	}))

	c := &configuration.Config{
		Show:       showConfig,
		Movie:      movieConfig,
		File:       fileConfig,
		Library:    config,
		Downloader: downloaderConfig,
	}

	return &mockLibrary{
		Library:    New(c),
		tmpDir:     tmpDir,
		httpServer: ts,
	}, nil
}
