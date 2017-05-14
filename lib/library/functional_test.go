package library

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/odwrtw/polochon/lib"
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
	if filepath.HasPrefix(m.tmpDir, os.TempDir()) {
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
	detailer, err := polochon.ConfigureDetailer("mock", nil)
	if err != nil {
		return nil, err
	}

	// Get the mock subtitler
	subtitler, err := polochon.ConfigureSubtitler("mock", nil)
	if err != nil {
		return nil, err
	}

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

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "mockContent")
	}))

	c := &configuration.Config{
		Show:    showConfig,
		Movie:   movieConfig,
		File:    fileConfig,
		Library: config,
	}

	return &mockLibrary{
		Library:    New(c),
		tmpDir:     tmpDir,
		httpServer: ts,
	}, nil
}
