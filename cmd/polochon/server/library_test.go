package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gorilla/mux"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/odwrtw/polochon/lib/nfo"
)

func writeTestNFO(t *testing.T, path string, value any) {
	t.Helper()

	var content bytes.Buffer
	if err := nfo.Write(&content, value); err != nil {
		t.Fatalf("write NFO %q: %v", path, err)
	}
	if err := os.WriteFile(path, content.Bytes(), 0o644); err != nil {
		t.Fatalf("write NFO file %q: %v", path, err)
	}
}

func newLibraryServer(t *testing.T) *Server {
	t.Helper()

	root := t.TempDir()
	movieDir := filepath.Join(root, "movies")
	showDir := filepath.Join(root, "shows")
	for _, dir := range []string{movieDir, showDir} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("create library directory %q: %v", dir, err)
		}
	}

	moviePath := filepath.Join(movieDir, "Movie (2001)", "movie.mp4")
	if err := os.Mkdir(filepath.Dir(moviePath), 0o755); err != nil {
		t.Fatalf("create movie directory: %v", err)
	}
	if err := os.WriteFile(moviePath, nil, 0o644); err != nil {
		t.Fatalf("create movie: %v", err)
	}
	movie := &polochon.Movie{
		BaseVideo: polochon.BaseVideo{VideoMetadata: polochon.VideoMetadata{
			Quality: polochon.Quality1080p,
		}},
		ImdbID:        "tt0000001",
		Title:         "Movie",
		OriginalTitle: "Original Movie",
		Plot:          "Movie plot",
		Runtime:       120,
		Genres:        []string{"Drama"},
		Year:          2001,
	}
	writeTestNFO(t, polochon.NewFile(moviePath).NfoPath(), movie)

	showPath := filepath.Join(showDir, "Show")
	seasonPath := filepath.Join(showPath, "Season 1")
	if err := os.MkdirAll(seasonPath, 0o755); err != nil {
		t.Fatalf("create show directory: %v", err)
	}
	show := &polochon.Show{
		ImdbID: "tt0000002",
		Title:  "Show",
		Plot:   "Show plot",
		TvdbID: 2,
		URL:    "https://provider.example/episode-guide",
		Year:   2002,
	}
	writeTestNFO(t, filepath.Join(showPath, "tvshow.nfo"), show)

	episodePath := filepath.Join(seasonPath, "episode.mp4")
	if err := os.WriteFile(episodePath, nil, 0o644); err != nil {
		t.Fatalf("create episode: %v", err)
	}
	episode := &polochon.ShowEpisode{
		BaseVideo: polochon.BaseVideo{VideoMetadata: polochon.VideoMetadata{
			Quality: polochon.Quality720p,
		}},
		ShowImdbID:    show.ImdbID,
		ShowTitle:     show.Title,
		ShowTvdbID:    show.TvdbID,
		Season:        1,
		Episode:       1,
		Title:         "Episode",
		Plot:          "Episode plot",
		Runtime:       42,
		EpisodeImdbID: "tt0000003",
	}
	writeTestNFO(t, polochon.NewFile(episodePath).NfoPath(), episode)

	config := &configuration.Config{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		File: polochon.FileConfig{
			VideoExtensions: []string{".mp4"},
		},
		Library: configuration.LibraryConfig{
			MovieDir: movieDir,
			ShowDir:  showDir,
		},
	}
	lib := library.New(config)
	if err := lib.RebuildIndex(); err != nil {
		t.Fatalf("rebuild index: %v", err)
	}

	// Library responses must use the in-memory index, not read NFO files.
	for _, path := range []string{
		polochon.NewFile(moviePath).NfoPath(),
		filepath.Join(showPath, "tvshow.nfo"),
		polochon.NewFile(episodePath).NfoPath(),
	} {
		if err := os.Remove(path); err != nil {
			t.Fatalf("remove NFO %q: %v", path, err)
		}
	}

	srv, err := New(config, lib, "", config.Logger)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return srv
}

func TestLibraryResponsesUseCachedNFOMetadata(t *testing.T) {
	srv := newLibraryServer(t)

	t.Run("movies", func(t *testing.T) {
		rr := httptest.NewRecorder()
		srv.movieIndex(rr, httptest.NewRequest(http.MethodGet, "/movies", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}

		var movies map[string]*polochon.Movie
		if err := json.Unmarshal(rr.Body.Bytes(), &movies); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		movie := movies["tt0000001"]
		if movie == nil {
			t.Fatal("movie missing from response")
		}
		if movie.OriginalTitle != "Original Movie" || movie.Plot != "Movie plot" || movie.Runtime != 120 || len(movie.Genres) != 1 {
			t.Fatalf("movie NFO metadata was not cached: %+v", movie)
		}
	})

	t.Run("movie details", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/movies/tt0000001", nil), map[string]string{"id": "tt0000001"})
		srv.getMovieDetails(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}

		var movie polochon.Movie
		if err := json.Unmarshal(rr.Body.Bytes(), &movie); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if movie.Plot != "Movie plot" {
			t.Fatalf("movie detail did not use cached NFO metadata: %+v", movie)
		}
	})

	t.Run("shows", func(t *testing.T) {
		rr := httptest.NewRecorder()
		srv.showIds(rr, httptest.NewRequest(http.MethodGet, "/shows", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}

		var shows map[string]struct {
			*polochon.Show
			Seasons map[string]map[string]*polochon.ShowEpisode `json:"seasons"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &shows); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if bytes.Contains(rr.Body.Bytes(), []byte(`"url"`)) {
			t.Fatal("show episode-guide URL must remain excluded from JSON")
		}
		show := shows["tt0000002"]
		if show.Show == nil || show.Plot != "Show plot" || show.TvdbID != 2 {
			t.Fatalf("show NFO metadata was not cached: %+v", show.Show)
		}
		episode := show.Seasons["01"]["01"]
		if episode == nil || episode.Title != "Episode" || episode.Plot != "Episode plot" || episode.Runtime != 42 || episode.EpisodeImdbID != "tt0000003" {
			t.Fatalf("episode NFO metadata was not cached: %+v", episode)
		}
	})

	t.Run("episode details", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/shows/tt0000002/seasons/1/episodes/1", nil), map[string]string{
			"id": "tt0000002", "season": "1", "episode": "1",
		})
		srv.getShowEpisodeIDDetails(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}

		var episode polochon.ShowEpisode
		if err := json.Unmarshal(rr.Body.Bytes(), &episode); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if episode.Plot != "Episode plot" {
			t.Fatalf("episode detail did not use cached NFO metadata: %+v", episode)
		}
	})
}

func TestLibraryShowJSONContract(t *testing.T) {
	srv := newLibraryServer(t)

	expectedShowFields := map[string]struct{}{
		"title": {}, "rating": {}, "plot": {}, "tvdb_id": {}, "imdb_id": {},
		"year": {}, "first_aired": {},
		"fanart_file": {}, "banner_file": {}, "poster_file": {}, "nfo_file": {},
	}
	expectedEpisodeFields := map[string]struct{}{
		"title": {}, "show_title": {}, "season": {}, "episode": {},
		"tvdb_id": {}, "aired": {}, "plot": {}, "runtime": {}, "thumb": {},
		"rating": {}, "show_imdb_id": {}, "show_tvdb_id": {}, "imdb_id": {},
		"date_added": {}, "quality": {}, "release_group": {}, "audio_codec": {},
		"video_codec": {}, "container": {}, "embedded_subtitles": {},
		"torrents": {}, "filename": {}, "size": {}, "subtitles": {}, "nfo_file": {},
	}

	t.Run("index", func(t *testing.T) {
		rr := httptest.NewRecorder()
		srv.showIds(rr, httptest.NewRequest(http.MethodGet, "/shows", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}

		var shows map[string]map[string]json.RawMessage
		if err := json.Unmarshal(rr.Body.Bytes(), &shows); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		show := shows["tt0000002"]
		if show == nil {
			t.Fatal("show missing from response")
		}

		// The seasons field is present with string-formatted numeric keys.
		rawSeasons, ok := show["seasons"]
		if !ok {
			t.Fatal("show response missing seasons field")
		}
		var seasons map[string]map[string]json.RawMessage
		if err := json.Unmarshal(rawSeasons, &seasons); err != nil {
			t.Fatalf("decode seasons: %v", err)
		}
		rawEpisode := seasons["01"]["01"]
		if rawEpisode == nil {
			t.Fatal("episode missing from seasons")
		}
		var episode map[string]json.RawMessage
		if err := json.Unmarshal(rawEpisode, &episode); err != nil {
			t.Fatalf("decode episode: %v", err)
		}
		assertExactFields(t, episode, expectedEpisodeFields)

		delete(show, "seasons")
		assertExactFields(t, show, expectedShowFields)
		if got := string(show["title"]); got != `"Show"` {
			t.Fatalf("title = %s, want %q", got, `"Show"`)
		}
		if _, excluded := show["url"]; excluded {
			t.Fatal("URL field must remain excluded from JSON")
		}
		for _, field := range []string{"banner", "fanart", "poster"} {
			if _, excluded := show[field]; excluded {
				t.Fatalf("%s must remain excluded from JSON", field)
			}
		}
	})

	t.Run("details", func(t *testing.T) {
		rr := httptest.NewRecorder()
		req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/shows/tt0000002", nil), map[string]string{"id": "tt0000002"})
		srv.getShowDetails(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}

		var show map[string]json.RawMessage
		if err := json.Unmarshal(rr.Body.Bytes(), &show); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		var seasons map[string]map[string]json.RawMessage
		if err := json.Unmarshal(show["seasons"], &seasons); err != nil {
			t.Fatalf("decode seasons: %v", err)
		}
		if _, ok := seasons["01"]["01"]; !ok {
			t.Fatal("episode missing from seasons")
		}
		delete(show, "seasons")
		assertExactFields(t, show, expectedShowFields)
	})
}

func TestLibraryJSONIsBackwardCompatibleWithMaster(t *testing.T) {
	srv := newLibraryServer(t)

	// These snapshots are the fields and values emitted by master before the
	// metadata was added to the indexes. date_added is checked with a marker
	// because the test NFO writer timestamps it at runtime; the marker verifies
	// that the field remains present and remains a string.
	legacyMovie := `{"tt0000001":{"date_added":"<runtime>","quality":"1080p","release_group":"","audio_codec":"","video_codec":"","container":"","embedded_subtitles":null,"filename":"movie.mp4","title":"Movie","year":2001,"size":0,"subtitles":[],"fanart_file":null,"thumb_file":null,"nfo_file":{"name":"movie.nfo","size":595}}}`
	legacyShow := `{"tt0000002":{"title":"Show","fanart_file":null,"banner_file":null,"poster_file":null,"nfo_file":{"name":"tvshow.nfo","size":301},"seasons":{"01":{"01":{"date_added":"<runtime>","quality":"720p","release_group":"","audio_codec":"","video_codec":"","container":"","embedded_subtitles":null,"filename":"episode.mp4","size":0,"subtitles":null,"nfo_file":{"name":"episode.nfo","size":635}}}}}}`
	legacyMovieDetail := `{"date_added":"<runtime>","quality":"1080p","release_group":"","audio_codec":"","video_codec":"","container":"","embedded_subtitles":null,"filename":"movie.mp4","size":0,"subtitles":[],"fanart_file":null,"thumb_file":null,"nfo_file":{"name":"movie.nfo","size":595}}`
	legacyShowDetail := `{"title":"Show","fanart_file":null,"banner_file":null,"poster_file":null,"nfo_file":{"name":"tvshow.nfo","size":301},"seasons":{"01":{"01":{"date_added":"<runtime>","quality":"720p","release_group":"","audio_codec":"","video_codec":"","container":"","embedded_subtitles":null,"filename":"episode.mp4","size":0,"subtitles":null,"nfo_file":{"name":"episode.nfo","size":635}}}}}`
	legacySeasonDetail := `{"show_imdb_id":"tt0000002","season":1,"episodes":{"1":{"date_added":"<runtime>","quality":"720p","release_group":"","audio_codec":"","video_codec":"","container":"","embedded_subtitles":null,"filename":"episode.mp4","size":0,"subtitles":null,"nfo_file":{"name":"episode.nfo","size":635}}}}`
	legacyEpisodeDetail := `{"date_added":"<runtime>","quality":"720p","release_group":"","audio_codec":"","video_codec":"","container":"","embedded_subtitles":null,"filename":"episode.mp4","size":0,"subtitles":null,"nfo_file":{"name":"episode.nfo","size":635}}`

	decode := func(t *testing.T, body []byte) any {
		t.Helper()
		var value any
		if err := json.Unmarshal(body, &value); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return value
	}
	legacy := func(t *testing.T, snapshot string) any {
		t.Helper()
		return decode(t, []byte(snapshot))
	}
	request := func(t *testing.T, fn http.HandlerFunc, path string, vars map[string]string) any {
		t.Helper()
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if vars != nil {
			req = mux.SetURLVars(req, vars)
		}
		fn(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		return decode(t, rr.Body.Bytes())
	}

	assertJSONSubset(t, legacy(t, legacyMovie), request(t, srv.movieIndex, "/movies", nil), "movies")
	assertJSONSubset(t, legacy(t, legacyShow), request(t, srv.showIds, "/shows", nil), "shows")
	assertJSONSubset(t, legacy(t, legacyMovieDetail), request(t, srv.getMovieDetails, "/movies/tt0000001", map[string]string{"id": "tt0000001"}), "movie detail")
	assertJSONSubset(t, legacy(t, legacyShowDetail), request(t, srv.getShowDetails, "/shows/tt0000002", map[string]string{"id": "tt0000002"}), "show detail")
	assertJSONSubset(t, legacy(t, legacySeasonDetail), request(t, srv.getSeasonDetails, "/shows/tt0000002/seasons/1", map[string]string{"id": "tt0000002", "season": "1"}), "season detail")
	assertJSONSubset(t, legacy(t, legacyEpisodeDetail), request(t, srv.getShowEpisodeIDDetails, "/shows/tt0000002/seasons/1/episodes/1", map[string]string{
		"id": "tt0000002", "season": "1", "episode": "1",
	}), "episode detail")
}

func assertJSONSubset(t *testing.T, want, got any, path string) {
	t.Helper()

	if wantString, ok := want.(string); ok && wantString == "<runtime>" {
		if _, ok := got.(string); !ok {
			t.Fatalf("%s changed type: got %#v, want string", path, got)
		}
		return
	}

	wantObject, wantIsObject := want.(map[string]any)
	if !wantIsObject {
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("%s changed: got %#v, want %#v", path, got, want)
		}
		return
	}

	gotObject, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("%s changed type: got %#v, want object", path, got)
	}
	for field, wantValue := range wantObject {
		gotValue, ok := gotObject[field]
		if !ok {
			t.Fatalf("%s.%s was removed", path, field)
		}
		assertJSONSubset(t, wantValue, gotValue, path+"."+field)
	}
}

func TestLibraryEpisodeJSONContract(t *testing.T) {
	srv := newLibraryServer(t)

	expectedEpisodeFields := map[string]struct{}{
		"title": {}, "show_title": {}, "season": {}, "episode": {},
		"tvdb_id": {}, "aired": {}, "plot": {}, "runtime": {}, "thumb": {},
		"rating": {}, "show_imdb_id": {}, "show_tvdb_id": {}, "imdb_id": {},
		"date_added": {}, "quality": {}, "release_group": {}, "audio_codec": {},
		"video_codec": {}, "container": {}, "embedded_subtitles": {},
		"torrents": {}, "filename": {}, "size": {}, "subtitles": {}, "nfo_file": {},
	}

	rr := httptest.NewRecorder()
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/shows/tt0000002/seasons/1/episodes/1", nil), map[string]string{
		"id": "tt0000002", "season": "1", "episode": "1",
	})
	srv.getShowEpisodeIDDetails(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var episode map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &episode); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertExactFields(t, episode, expectedEpisodeFields)

	for field, expected := range map[string]string{
		"title":        `"Episode"`,
		"plot":         `"Episode plot"`,
		"runtime":      `42`,
		"show_imdb_id": `"tt0000002"`,
		"quality":      `"720p"`,
		"filename":     `"episode.mp4"`,
		"season":       `1`,
		"episode":      `1`,
	} {
		if string(episode[field]) != expected {
			t.Errorf("%s = %s, want %s", field, episode[field], expected)
		}
	}

	// The episode must be flat: no nested ShowEpisode or VideoMetadata objects.
	for _, forbidden := range []string{"show_episode", "video_metadata", "base_video"} {
		if _, nested := episode[forbidden]; nested {
			t.Fatalf("episode JSON must not contain nested %q object", forbidden)
		}
	}
}

func assertExactFields(t *testing.T, obj map[string]json.RawMessage, expected map[string]struct{}) {
	t.Helper()
	got := map[string]struct{}{}
	for field := range obj {
		got[field] = struct{}{}
	}
	if len(got) != len(expected) {
		t.Fatalf("field count = %d, want %d\ngot:  %v\nwant: %v", len(got), len(expected), got, expected)
	}
	for field := range expected {
		if _, ok := got[field]; !ok {
			t.Fatalf("missing field %q; got %v", field, got)
		}
	}
}
