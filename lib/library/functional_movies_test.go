package library

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/odwrtw/polochon/lib"
	_ "github.com/odwrtw/polochon/modules/mock"
)

func (m *mockLibrary) mockMovie(name string) (*polochon.Movie, error) {
	path := filepath.Join(m.tmpDir, "downloads", name)

	// Create the movie file
	if _, err := os.Create(path); err != nil {
		return nil, err
	}

	movie := polochon.NewMovie(m.movieConfig)
	movie.Fanart = m.httpServer.URL
	movie.Thumb = m.httpServer.URL
	movie.Path = filepath.Join(m.tmpDir, "downloads", name)

	if err := movie.GetDetails(mockLogEntry); err != nil {
		return nil, err
	}

	return movie, nil
}

func TestAddMovie(t *testing.T) {
	lib, err := newMockLibrary()
	defer lib.cleanup()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	m, err := lib.mockMovie("movieTest.mp4")
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	oldMoviePath := m.Path

	// Add the movie to the library
	if err := lib.Add(m, mockLogEntry); err != nil {
		t.Fatalf("failed to add the movie: %q", err)
	}

	// Check the new file location
	expectedPath := filepath.Join(lib.tmpDir, "movies/Movie tt12345 (2000)/movieTest.mp4")
	if m.Path != expectedPath {
		t.Errorf("file location, expected %q got %q", expectedPath, m.Path)
	}

	// Check that the old path is a symlink that point to the movie's new path
	gotNewPath, err := filepath.EvalSymlinks(oldMoviePath)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if gotNewPath != m.Path {
		t.Errorf("invalid symlink, expected %q got %q", m.Path, gotNewPath)
	}

	// Check the content of the downloaded files
	for _, imgPath := range []string{
		m.MovieFanartPath(),
		m.MovieThumbPath(),
	} {
		content, err := ioutil.ReadFile(imgPath)
		if err != nil {
			t.Fatalf("failed to add the movie: %q", err)
		}

		// The mock content comes from the httptest server
		if string(content) != "mockContent" {
			t.Error("invalid image content")
		}
	}

	// Check the movie index
	expectedIDs := []string{m.ImdbID}
	gotIDs, err := lib.MovieIDs()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	if !reflect.DeepEqual(gotIDs, expectedIDs) {
		t.Errorf("invalid ids, expected %+v got %+v", expectedIDs, gotIDs)
	}

	// Get the movie from the lib
	movieFromLib, err := lib.GetMovie(m.ImdbID)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	if !reflect.DeepEqual(m, movieFromLib) {
		t.Errorf("invalid movie from lib, expected %+v got %+v", m, movieFromLib)
	}

	// Rebuild the index, the movie should be found and added to the index
	if err := lib.RebuildIndex(mockLogEntry); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	// Check the movie index again
	gotIDs, err = lib.MovieIDs()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	if !reflect.DeepEqual(gotIDs, expectedIDs) {
		t.Errorf("invalid ids, expected %+v got %+v", expectedIDs, gotIDs)
	}
}

func TestDeleteMovie(t *testing.T) {
	lib, err := newMockLibrary()
	defer lib.cleanup()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	m, err := lib.mockMovie("movieTest")
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	// Add the movie to the library
	if err := lib.Add(m, mockLogEntry); err != nil {
		t.Fatalf("failed to add the movie: %q", err)
	}

	// Count the movies in the index
	ids, err := lib.MovieIDs()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	movieCount := len(ids)
	if movieCount != 1 {
		t.Fatalf("the library should contains %d movie instead of 1", movieCount)
	}

	// Delete the movie from the library
	if err := lib.Delete(m, mockLogEntry); err != nil {
		t.Fatalf("failed to add the movie: %q", err)
	}

	// Count the movies in the index
	ids, err = lib.MovieIDs()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	movieCount = len(ids)
	if movieCount != 0 {
		t.Fatalf("the library should contains %d movie instead of 0", movieCount)
	}

	// Ensure the movie folder has been deleted
	if exists(lib.getMovieDir(m)) {
		t.Fatal("the movie directory should have been deleted", err)
	}
}
