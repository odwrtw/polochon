package library

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
	_ "github.com/odwrtw/polochon/modules/mock"
)

func (m *mockLibrary) mockEpisode(s *polochon.Show, name string) (*polochon.ShowEpisode, error) {
	path := filepath.Join(m.tmpDir, "downloads", name)

	// Create the episode file
	if _, err := os.Create(path); err != nil {
		return nil, err
	}

	e := polochon.NewShowEpisode(m.showConfig)
	e.Path = filepath.Join(m.tmpDir, "downloads", name)
	e.Thumb = m.httpServer.URL
	e.Show = s

	if err := polochon.GetDetails(e, mockLogEntry); err != nil {
		return nil, err
	}

	if err := polochon.GetSubtitles(e, m.SubtitleLanguages, mockLogEntry); err != nil {
		return nil, err
	}

	return e, nil
}

func (m *mockLibrary) mockShow() (*polochon.Show, error) {
	s := polochon.NewShow(m.showConfig)

	// Set the images URLs
	s.Banner = m.httpServer.URL
	s.Fanart = m.httpServer.URL
	s.Poster = m.httpServer.URL
	s.ImdbID = "tt12345"

	if err := polochon.GetDetails(s, mockLogEntry); err != nil {
		return nil, err
	}

	return s, nil
}

func TestAddEpisode(t *testing.T) {
	lib, err := newMockLibrary()
	defer lib.cleanup()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	// Get a mock show
	show, err := lib.mockShow()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	// The mock detailer fakes episodes, let's remove them
	show.Episodes = nil

	// Get a mock episode
	episode, err := lib.mockEpisode(show, "episodeTest.mp4")
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	oldEpisodePath := episode.Path

	// Add the episode to the library
	if err := lib.Add(episode, mockLogEntry); err != nil {
		t.Fatalf("failed to add the episode: %q", err)
	}

	// Check the content of the downloaded subtitles files
	for _, lang := range lib.SubtitleLanguages {
		sub := lib.GetSubtitle(episode, lang)
		if sub == nil {
			t.Fatal("should have subtitle")
		}
		content, err := ioutil.ReadFile(episode.SubtitlePath(lang))
		if err != nil {
			t.Fatalf("failed to read the episode's subtitle : %q", err)
		}

		// The mock content comes from the httptest server
		if string(content) != fmt.Sprintf("subtitle in %s", lang) {
			t.Error("invalid subtitle content")
		}
	}

	// Check the new file location
	expectedPath := filepath.Join(lib.tmpDir, "shows/Show tt12345/Season 1/episodeTest.mp4")
	if episode.Path != expectedPath {
		t.Errorf("file location, expected %q got %q", expectedPath, episode.Path)
	}

	// Check that the old path is a symlink that point to the episode's new path
	gotNewPath, err := filepath.EvalSymlinks(oldEpisodePath)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if gotNewPath != episode.Path {
		t.Errorf("invalid symlink, expected %q got %q", episode.Path, gotNewPath)
	}

	// Get a new mock episode
	episode, err = lib.mockEpisode(show, "episodeTest.mp4")
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	// Add the same episode again, this should replace the old one
	if err := lib.Add(episode, mockLogEntry); err != nil {
		t.Fatalf("failed to add the episode again: %q", err)
	}

	// Test the show content
	testShow(t, episode, lib)

	// Test the season
	testSeason(t, episode, lib)

	episodeFromLib, err := lib.GetEpisode(episode.ShowImdbID, episode.Season, episode.Episode)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	// The show is not retrieved by get the GetEpisode method, let's add it
	// manually
	episodeFromLib.Show = episode.Show

	// The mock episode have the data but not the episode from the lib, let's
	// remove the data to compare the two
	for _, s := range episode.Subtitles {
		s.Data = nil
	}

	if !reflect.DeepEqual(episode, episodeFromLib) {
		t.Errorf("invalid episode from lib, expected %+v got %+v", episode, episodeFromLib)
	}

	// Expected indexed season
	expectedIndexedSeason := &index.Season{
		Path: filepath.Join(lib.tmpDir, "shows/Show tt12345/Season 1"),
		Episodes: map[int]*index.Episode{
			1: {
				Path:          filepath.Join(lib.tmpDir, "shows/Show tt12345/Season 1/episodeTest.mp4"),
				Filename:      "episodeTest.mp4",
				VideoMetadata: episode.VideoMetadata,
				Subtitles: []*index.Subtitle{
					{Lang: polochon.FR, Size: 17},
					{Lang: polochon.EN, Size: 17},
				},
			},
		},
	}

	// Expected indexed show
	expectedIndexedShow := &index.Show{
		Path: filepath.Join(lib.tmpDir, "shows/Show tt12345"),
		Seasons: map[int]*index.Season{
			1: expectedIndexedSeason,
		},
		Title: "Show tt12345",
	}

	// Expected IDs
	expectedIDs := map[string]*index.Show{
		"tt12345": expectedIndexedShow,
	}

	// Ensure the index if valid
	gotIDs := lib.ShowIDs()
	if !reflect.DeepEqual(expectedIDs, gotIDs) {
		t.Fatalf("invalid show ids, expected %#v got %#v", expectedIDs, gotIDs)
	}

	// Ensure the library has the show episode
	hasEpisode, err := lib.HasVideo(episode)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if !hasEpisode {
		t.Fatal("the episode should be in the index")
	}

	// Get the indexed show
	gotIndexedShow, err := lib.GetIndexedShow(episode.ShowImdbID)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	if !reflect.DeepEqual(expectedIndexedShow, gotIndexedShow) {
		t.Fatalf("invalid show ids, expected %+v got %+v", expectedIndexedShow, gotIndexedShow)
	}

	// Get the indexed season
	gotIndexedSeason, err := lib.GetIndexedSeason(episode.ShowImdbID, episode.Season)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	if !reflect.DeepEqual(expectedIndexedSeason, gotIndexedSeason) {
		t.Fatalf("invalid season, expected %+v got %+v", expectedIndexedSeason, gotIndexedSeason)
	}

	// Rebuild the index, the episode should be found and added to the index
	if err := lib.RebuildIndex(mockLogEntry); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	// Ensure the index is still valid after a rebuild
	gotIDs = lib.ShowIDs()
	if !reflect.DeepEqual(expectedIDs, gotIDs) {
		t.Fatalf("invalid show ids, expected %+v got %+v", expectedIDs, gotIDs)
	}
}

func testShow(t *testing.T, episode *polochon.ShowEpisode, lib *mockLibrary) {
	// Check the content of the downloaded images of the show
	for _, name := range []string{
		"banner.jpg",
		"fanart.jpg",
		"poster.jpg",
	} {
		path := filepath.Join(lib.getShowDir(episode), name)
		content, err := ioutil.ReadFile(path)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}

		// The mock content comes from the httptest server
		if string(content) != "mockContent" {
			t.Error("invalid image content")
		}
	}

	// Get the show from the library
	showFromLib, err := lib.GetShow(episode.ShowImdbID)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	// The images URL are not stored in the NFO, maybe they should...
	showFromLib.Banner = lib.httpServer.URL
	showFromLib.Fanart = lib.httpServer.URL
	showFromLib.Poster = lib.httpServer.URL

	if !reflect.DeepEqual(episode.Show, showFromLib) {
		t.Errorf("invalid show from lib, expected %+v got %+v", episode.Show, showFromLib)
	}
}

func testSeason(t *testing.T, episode *polochon.ShowEpisode, lib *mockLibrary) {
	// Get the season from the library
	seasonFromLib, err := lib.GetSeason(episode.ShowImdbID, episode.Season)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	expected := polochon.NewShowSeason(lib.showConfig)
	expected.Season = episode.Season
	expected.ShowImdbID = episode.ShowImdbID

	if !reflect.DeepEqual(seasonFromLib, expected) {
		t.Errorf("invalid show from lib, expected %+v got %+v", expected, seasonFromLib)
	}
}

func TestDeleteEpisode(t *testing.T) {
	lib, err := newMockLibrary()
	defer lib.cleanup()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	// Get a mock show
	show, err := lib.mockShow()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	// The mock detailer fakes episodes, let's remove them
	show.Episodes = nil

	// Get a mock episode
	episode, err := lib.mockEpisode(show, "episodeTest.mp4")
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	// Add the episode to the library
	if err := lib.Add(episode, mockLogEntry); err != nil {
		t.Fatalf("failed to add the episode: %q", err)
	}

	// Add the episode to the library
	if err := lib.Delete(episode, mockLogEntry); err != nil {
		t.Fatalf("failed to remove the episode: %q", err)
	}

	// Ensure the index if valid
	gotIDs := lib.ShowIDs()
	expectedIDs := map[string]*index.Show{}

	if !reflect.DeepEqual(expectedIDs, gotIDs) {
		t.Errorf("invalid show ids, expected %+v got %+v", expectedIDs, gotIDs)
	}

	// Rebuild the index
	if err := lib.RebuildIndex(mockLogEntry); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	// Ensure the index is still valid after a rebuild
	gotIDs = lib.ShowIDs()
	if !reflect.DeepEqual(expectedIDs, gotIDs) {
		t.Errorf("invalid show ids, expected %+v got %+v", expectedIDs, gotIDs)
	}
}
