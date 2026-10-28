package simkl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
)

func TestRecordedPublicSchemas(t *testing.T) {
	for _, filename := range []string{"movies-trending.json", "tv-trending.json", "movie.json", "tv.json"} {
		data, err := os.ReadFile(filepath.Join("testdata", filename))
		if err != nil {
			t.Fatal(err)
		}
		var records []record
		if strings.Contains(filename, "trending") {
			err = json.Unmarshal(data, &records)
		} else {
			var r record
			err = json.Unmarshal(data, &r)
			records = []record{r}
		}
		if err != nil {
			t.Fatalf("%s: %v", filename, err)
		}
		kind := "movies"
		if strings.HasPrefix(filename, "tv") {
			kind = "tv"
		}
		for _, r := range records {
			if !r.compatible(kind) {
				t.Fatalf("recorded %s record is unrepresentable", filename)
			}
			if kind == "movies" {
				m := &polochon.Movie{}
				r.movie(m)
				if m.Year == 0 || m.Runtime == 0 {
					t.Fatalf("lost runtime/year: %+v", m)
				}
			}
		}
	}
	data, err := os.ReadFile("testdata/episodes.json")
	if err != nil {
		t.Fatal(err)
	}
	var episodes []episodeRecord
	if err = json.Unmarshal(data, &episodes); err != nil {
		t.Fatal(err)
	}
	if len(episodes) != 3 || episodes[0].Season.integer() != 1 || episodes[0].Episode.integer() != 1 || episodes[0].IDs.simkl() == "" {
		t.Fatal("recorded TV episode schema changed")
	}
}

func TestActualConfigurationLoaderWithoutModuleParams(t *testing.T) {
	dir := t.TempDir()
	raw := fmt.Sprintf("movie:\n  dir: %q\n  explorers: [simkl]\n  detailers: [simkl]\n  searchers: [simkl]\nshow:\n  dir: %q\n  explorers: [simkl]\n  detailers: [simkl]\n  searchers: [simkl]\n  calendar: simkl\nwishlist:\n  wishlisters: [simkl]\n", dir, dir)
	config, err := configuration.LoadConfig(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Movie.Explorers) != 1 || len(config.Show.Searchers) != 1 || len(config.Wishlist.Wishlisters) != 1 || config.Show.Calendar.Name() != "simkl" {
		t.Fatal("existing role configuration did not load")
	}
	if config.Movie.Explorers[0] != config.Show.Explorers[0] {
		t.Fatal("roles do not share module state")
	}
}

func TestCompatibleLibraryImportAndNFOReload(t *testing.T) {
	var art bytes.Buffer
	if err := jpeg.Encode(&art, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	s := testModule(t, Params{}, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/art/") {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(art.Bytes())
			return
		}
		base := "http://" + r.Host
		switch r.URL.Path {
		case "/movies/1":
			a := catalogRecord(1, "tt1", "movie")
			a.Title = "Fixture Movie"
			a.Poster = base + "/art/poster.jpg"
			a.Fanart = base + "/art/fanart.jpg"
			writeJSON(w, a)
		case "/tv/10":
			a := catalogRecord(10, "tt10", "show")
			a.Title = "Fixture Show"
			a.Poster = base + "/art/poster.jpg"
			a.Fanart = base + "/art/fanart.jpg"
			a.Banner = base + "/art/banner.jpg"
			writeJSON(w, a)
		case "/tv/episodes/10":
			writeJSON(w, []episodeRecord{{Title: "Pilot", Season: "1", Episode: "1", Date: "2020-01-01T00:00:00Z"}})
		default:
			t.Errorf("unexpected endpoint %s", r.URL.Path)
			w.WriteHeader(404)
		}
	})
	dir := t.TempDir()
	movieDir := filepath.Join(dir, "movies")
	showDir := filepath.Join(dir, "shows")
	for _, d := range []string{movieDir, showDir} {
		if err := os.Mkdir(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &configuration.Config{Logger: s.log, Library: configuration.LibraryConfig{MovieDir: movieDir, ShowDir: showDir}, File: polochon.FileConfig{VideoExtensions: []string{".mkv"}}}
	cfg.Movie.Detailers = []polochon.Detailer{s}
	cfg.Show.Detailers = []polochon.Detailer{s}
	cfg.Show.Calendar = s
	l := library.New(cfg)
	s.client.remember("movies", catalogRecord(1, "tt1", "movie"))
	s.client.remember("tv", catalogRecord(10, "tt10", "show"))
	movie := polochon.NewMovie(cfg.Movie)
	movie.ImdbID = "tt1"
	if err := s.GetDetails(context.Background(), movie); err != nil {
		t.Fatal(err)
	}
	movie.Path = filepath.Join(dir, "movie.mkv")
	if err := os.WriteFile(movie.Path, []byte("fixture video"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := l.AddMovie(movie); err != nil {
		t.Fatal(err)
	}
	ep := polochon.NewShowEpisode(cfg.Show)
	ep.ShowImdbID = "tt10"
	ep.Season = 1
	ep.Episode = 1
	if err := s.GetDetails(context.Background(), ep); err != nil {
		t.Fatal(err)
	}
	ep.Path = filepath.Join(dir, "episode.mkv")
	if err := os.WriteFile(ep.Path, []byte("fixture video"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := l.AddShowEpisode(ep); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		m, err := l.GetMovie("tt1")
		if err != nil || m.ImdbID != "tt1" || m.Title != "Fixture Movie" {
			t.Fatalf("movie reload: %+v %v", m, err)
		}
		e, err := l.GetEpisode("tt10", 1, 1)
		if err != nil || e.ShowImdbID != "tt10" || e.Title != "Pilot" {
			t.Fatalf("episode reload: %+v %v", e, err)
		}
		if err = l.RebuildIndex(); err != nil {
			t.Fatal(err)
		}
	}
}
