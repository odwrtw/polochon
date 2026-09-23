package x1337x

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestSearchTorrentsFallsBackAndKeepsHealthyEndpoint(t *testing.T) {
	failedCalls := 0
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		failedCalls++
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer failed.Close()

	healthyCalls := 0
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		healthyCalls++
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/"):
			_, _ = w.Write([]byte(searchPage))
		case strings.HasPrefix(r.URL.Path, "/torrent/1/"):
			_, _ = w.Write([]byte(`<a href="magnet:?xt=urn:btih:abc">Magnet</a>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer healthy.Close()

	x := &X1337{}
	if err := x.InitWithParams(&Params{URLs: []string{failed.URL, healthy.URL}}); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		torrents, err := x.SearchTorrents("Example Movie")
		if err != nil {
			t.Fatal(err)
		}
		if len(torrents) != 1 {
			t.Fatalf("expected one torrent, got %d", len(torrents))
		}
		if torrents[0].Result.URL != "magnet:?xt=urn:btih:abc" {
			t.Fatalf("unexpected magnet %q", torrents[0].Result.URL)
		}
	}

	if failedCalls != 1 {
		t.Fatalf("failed endpoint was called %d times, want 1", failedCalls)
	}
	if healthyCalls != 4 {
		t.Fatalf("healthy endpoint was called %d times, want 4", healthyCalls)
	}
}

func TestSearchTorrentsFallsBackFromEmptyResults(t *testing.T) {
	emptyCalls := 0
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		emptyCalls++
		_, _ = w.Write([]byte(`<table class="table-list"></table>`))
	}))
	defer empty.Close()
	healthy := newTorrentServer(t, searchPage)

	x := &X1337{}
	if err := x.InitWithParams(&Params{URLs: []string{empty.URL, healthy.URL}}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		torrents, err := x.SearchTorrents("Example Movie")
		if err != nil || len(torrents) != 1 {
			t.Fatalf("expected one torrent from fallback, got %d: %v", len(torrents), err)
		}
	}
	if emptyCalls != 1 {
		t.Fatalf("empty endpoint was called %d times, want 1", emptyCalls)
	}
}

func TestSearchTorrentsAllEndpointsEmpty(t *testing.T) {
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<table class="table-list"></table>`))
	}))
	defer empty.Close()
	x := &X1337{}
	if err := x.InitWithParams(&Params{URLs: []string{empty.URL}}); err != nil {
		t.Fatal(err)
	}
	torrents, err := x.SearchTorrents("missing")
	if err != nil || len(torrents) != 0 {
		t.Fatalf("expected empty search without error, got %d: %v", len(torrents), err)
	}
}

func TestSearchTorrentsUsesParsedQuality(t *testing.T) {
	for _, test := range []struct {
		name    string
		quality polochon.Quality
	}{
		{name: "Example.Movie.2024.2160p.BluRay.x264", quality: polochon.Quality2160p},
		{name: "Example.Movie.2024.720p.BluRay.x264", quality: polochon.Quality720p},
		{name: "Example.Movie.2024.1080pish.BluRay.x264"},
	} {
		t.Run(test.name, func(t *testing.T) {
			page := strings.Replace(searchPage, "Example.Movie.2024.1080p.BluRay.x264.YIFY", test.name, 1)
			server := newTorrentServer(t, page)
			x := &X1337{}
			if err := x.InitWithParams(&Params{URLs: []string{server.URL}}); err != nil {
				t.Fatal(err)
			}
			torrents, err := x.SearchTorrents("Example Movie")
			if err != nil || len(torrents) != 1 {
				t.Fatalf("expected one torrent, got %d: %v", len(torrents), err)
			}
			if torrents[0].Quality != test.quality {
				t.Fatalf("quality = %q, want %q", torrents[0].Quality, test.quality)
			}
		})
	}
}

func TestGetTorrentsFallsBackFromIneligibleMirror(t *testing.T) {
	firstDetails := 0
	firstCalls := 0
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstCalls++
		if strings.HasPrefix(r.URL.Path, "/torrent/") {
			firstDetails++
			_, _ = w.Write([]byte(`<a href="magnet:?xt=urn:btih:untrusted">Magnet</a>`))
			return
		}
		_, _ = w.Write([]byte(strings.Replace(searchPage, ">uploader</td>", ">impostor</td>", 1)))
	}))
	defer first.Close()
	second := newTorrentServer(t, searchPage)

	x := &X1337{}
	if err := x.InitWithParams(&Params{URLs: []string{first.URL, second.URL}, MovieUsers: []string{"uploader"}}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		movie := polochon.NewMovie(polochon.MovieConfig{})
		movie.Title, movie.Year = "Example Movie", 2024
		if err := x.GetTorrents(context.Background(), movie); err != nil {
			t.Fatal(err)
		}
		if len(movie.Torrents) != 1 || movie.Torrents[0].Result.UploadUser != "uploader" {
			t.Fatalf("expected trusted torrent from fallback, got %+v", movie.Torrents)
		}
	}
	if firstCalls != 1 || firstDetails != 0 {
		t.Fatalf("first endpoint: %d requests, %d detail requests; want 1, 0", firstCalls, firstDetails)
	}
}

func TestGetTorrentsSkipsIneligibleDetails(t *testing.T) {
	for _, test := range []struct {
		name string
		user string
	}{
		{name: "untrusted uploader", user: "impostor"},
		{name: "wrong year", user: "uploader"},
		{name: "unknown quality", user: "uploader"},
		{name: "false quality token", user: "uploader"},
	} {
		t.Run(test.name, func(t *testing.T) {
			row := strings.SplitN(strings.SplitN(searchPage, "<tr>", 2)[1], "</tr>", 2)[0]
			row = strings.Replace(row, "/torrent/1/", "/torrent/2/", 1)
			row = strings.Replace(row, ">uploader</td>", ">"+test.user+"</td>", 1)
			switch test.name {
			case "wrong year":
				row = strings.Replace(row, "Example.Movie.2024", "Example.Movie.2023", 1)
			case "unknown quality":
				row = strings.Replace(row, "2024.1080p.", "2024.unknown.", 1)
			case "false quality token":
				row = strings.Replace(row, "2024.1080p.", "2024.1080pish.", 1)
			}
			page := strings.Replace(searchPage, "<tr>", "<tr>"+row+"</tr><tr>", 1)
			ineligibleDetails := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case strings.HasPrefix(r.URL.Path, "/search/"):
					_, _ = w.Write([]byte(page))
				case strings.HasPrefix(r.URL.Path, "/torrent/2/"):
					ineligibleDetails++
					_, _ = w.Write([]byte(`<a href="magnet:?xt=urn:btih:unwanted">Magnet</a>`))
				case strings.HasPrefix(r.URL.Path, "/torrent/1/"):
					_, _ = w.Write([]byte(`<a href="magnet:?xt=urn:btih:abc">Magnet</a>`))
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)
			x := &X1337{}
			if err := x.InitWithParams(&Params{URLs: []string{server.URL}, MovieUsers: []string{"uploader"}}); err != nil {
				t.Fatal(err)
			}
			movie := polochon.NewMovie(polochon.MovieConfig{})
			movie.Title, movie.Year = "Example Movie", 2024
			if err := x.GetTorrents(context.Background(), movie); err != nil {
				t.Fatal(err)
			}
			if ineligibleDetails != 0 || len(movie.Torrents) != 1 || movie.Torrents[0].Result.URL != "magnet:?xt=urn:btih:abc" {
				t.Fatalf("fetched %d ineligible details; torrents: %+v", ineligibleDetails, movie.Torrents)
			}
		})
	}
}

func TestGetTorrentsSearchesMovieYear(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/search/Example%20Movie%202024/1/":
			_, _ = w.Write([]byte(searchPage))
		case "/torrent/1/Example-Movie-2024-1080p/":
			_, _ = w.Write([]byte(`<a href="magnet:?xt=urn:btih:abc">Magnet</a>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	x := &X1337{}
	if err := x.InitWithParams(&Params{URLs: []string{server.URL}, MovieUsers: []string{"uploader"}}); err != nil {
		t.Fatal(err)
	}
	movie := polochon.NewMovie(polochon.MovieConfig{})
	movie.Title, movie.Year = "Example Movie", 2024
	if err := x.GetTorrents(context.Background(), movie); err != nil || len(movie.Torrents) != 1 {
		t.Fatalf("year-specific search: %d torrents: %v", len(movie.Torrents), err)
	}
}

func TestGetTorrentsForMovie(t *testing.T) {
	server := newTorrentServer(t, searchPage)
	x := &X1337{}
	if err := x.InitWithParams(&Params{
		URLs:       []string{server.URL},
		MovieUsers: []string{" uploader "},
	}); err != nil {
		t.Fatal(err)
	}

	movie := polochon.NewMovie(polochon.MovieConfig{})
	movie.Title = "Example Movie"
	movie.Year = 2024
	movie.ImdbID = "tt1234567"
	if err := x.GetTorrents(context.Background(), movie); err != nil {
		t.Fatal(err)
	}

	if len(movie.Torrents) != 1 {
		t.Fatalf("expected one torrent, got %d", len(movie.Torrents))
	}
	torrent := movie.Torrents[0]
	if torrent.Type != polochon.TypeMovie || torrent.ImdbID != movie.ImdbID || torrent.Quality != polochon.Quality1080p {
		t.Fatalf("unexpected torrent %+v", torrent)
	}
}

func TestGetTorrentsMovieUsersAndYear(t *testing.T) {
	for _, test := range []struct {
		name  string
		user  string
		year  int
		title string
		want  bool
	}{
		{name: "untrusted uploader with spoofed release name", user: "other", year: 2024, title: "Example.Movie.2024.1080p.BluRay.x264.YIFY"},
		{name: "trusted uploader without release name", user: "uploader", year: 2024, title: "Example.Movie.2024.1080p.BluRay.x264", want: true},
		{name: "different year", user: "uploader", year: 2023, title: "Example.Movie.2024.1080p.BluRay.x264.YIFY"},
		{name: "unknown release year", user: "uploader", year: 2024, title: "Example.Movie.1080p.BluRay.x264.YIFY", want: true},
		{name: "unknown requested year", user: "uploader", title: "Example.Movie.2024.1080p.BluRay.x264.YIFY", want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			page := strings.Replace(searchPage, "Example.Movie.2024.1080p.BluRay.x264.YIFY", test.title, 1)
			page = strings.Replace(page, ">uploader</td>", ">"+test.user+"</td>", 1)
			server := newTorrentServer(t, page)
			x := &X1337{}
			if err := x.InitWithParams(&Params{URLs: []string{server.URL}, MovieUsers: []string{"uploader"}}); err != nil {
				t.Fatal(err)
			}
			movie := polochon.NewMovie(polochon.MovieConfig{})
			movie.Title, movie.Year = "Example Movie", test.year
			err := x.GetTorrents(context.Background(), movie)
			if test.want {
				if err != nil || len(movie.Torrents) != 1 {
					t.Fatalf("expected one torrent, got %d: %v", len(movie.Torrents), err)
				}
			} else if !errors.Is(err, polochon.ErrTorrentNotFound) || len(movie.Torrents) != 0 {
				t.Fatalf("expected no torrent, got %d: %v", len(movie.Torrents), err)
			}
		})
	}
}

func TestGetTorrentsPrefersTrustedUploaderOverSeeders(t *testing.T) {
	attackerRow := strings.Replace(searchPage, "/torrent/1/", "/torrent/2/", 1)
	attackerRow = strings.Replace(attackerRow, ">1,234</td>", ">99,999</td>", 1)
	attackerRow = strings.Replace(attackerRow, ">uploader</td>", ">impostor</td>", 1)
	row := strings.SplitN(strings.SplitN(attackerRow, "<tr>", 2)[1], "</tr>", 2)[0]
	page := strings.Replace(searchPage, "</tr>", "</tr><tr>"+row+"</tr>", 1)
	server := newTorrentServer(t, page)
	x := &X1337{}
	if err := x.InitWithParams(&Params{URLs: []string{server.URL}, MovieUsers: []string{"uploader"}}); err != nil {
		t.Fatal(err)
	}
	movie := polochon.NewMovie(polochon.MovieConfig{})
	movie.Title, movie.Year = "Example Movie", 2024
	if err := x.GetTorrents(context.Background(), movie); err != nil {
		t.Fatal(err)
	}
	if len(movie.Torrents) != 1 || movie.Torrents[0].Result.UploadUser != "uploader" {
		t.Fatalf("expected trusted uploader only, got %+v", movie.Torrents)
	}
}

func TestGetTorrentsForShowEpisode(t *testing.T) {
	page := strings.Replace(searchPage, "Example.Movie.2024.1080p.BluRay.x264.YIFY", "Example.Show.S01E02.1080p.WEB-DL.YIFY", 1)
	server := newTorrentServer(t, page)
	x := &X1337{}
	if err := x.InitWithParams(&Params{URLs: []string{server.URL}, ShowUsers: []string{"uploader"}}); err != nil {
		t.Fatal(err)
	}
	episode := polochon.NewShowEpisode(polochon.ShowConfig{})
	episode.ShowTitle, episode.Season, episode.Episode = "Example Show", 1, 2
	episode.ShowImdbID = "tt1234567"
	if err := x.GetTorrents(context.Background(), episode); err != nil {
		t.Fatal(err)
	}
	if len(episode.Torrents) != 1 || episode.Torrents[0].Type != polochon.TypeEpisode ||
		episode.Torrents[0].Season != 1 || episode.Torrents[0].Episode != 2 || episode.Torrents[0].ImdbID != episode.ShowImdbID {
		t.Fatalf("unexpected episode torrents: %+v", episode.Torrents)
	}

	movie := polochon.NewMovie(polochon.MovieConfig{})
	movie.Title = "Example Show"
	if err := x.GetTorrents(context.Background(), movie); !errors.Is(err, polochon.ErrTorrentNotFound) {
		t.Fatalf("expected no movie torrents without movie users, got %v", err)
	}
}

func TestGetTorrentsWithoutTrustedUsers(t *testing.T) {
	x := &X1337{}
	if err := x.InitWithParams(&Params{URLs: []string{"https://1337x.example"}}); err != nil {
		t.Fatal(err)
	}
	movie := polochon.NewMovie(polochon.MovieConfig{})
	movie.Title = "Example Movie"
	if err := x.GetTorrents(context.Background(), movie); !errors.Is(err, polochon.ErrTorrentNotFound) {
		t.Fatalf("expected no torrents without trusted users, got %v", err)
	}
}

func newTorrentServer(t *testing.T, page string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/"):
			_, _ = w.Write([]byte(page))
		case strings.HasPrefix(r.URL.Path, "/torrent/"):
			_, _ = w.Write([]byte(`<a href="magnet:?xt=urn:btih:abc">Magnet</a>`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestInitRejectsMissingOrInvalidEndpoints(t *testing.T) {
	for _, test := range []struct {
		name string
		urls []string
		want error
	}{
		{name: "missing", want: ErrMissingURLs},
		{name: "invalid scheme", urls: []string{"ftp://example.test"}, want: ErrInvalidEndpoint},
		{name: "query", urls: []string{"https://example.test/?q=1"}, want: ErrInvalidEndpoint},
	} {
		t.Run(test.name, func(t *testing.T) {
			x := &X1337{}
			err := x.InitWithParams(&Params{URLs: test.urls})
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
}

func TestParseSearchResults(t *testing.T) {
	results, err := parseSearchResults([]byte(searchPage))
	if err != nil {
		t.Fatal(err)
	}
	want := []searchResult{{
		Name: "Example.Movie.2024.1080p.BluRay.x264.YIFY", Detail: "/torrent/1/Example-Movie-2024-1080p/",
		Seeders: 1234, Leechers: 56, Size: 1500000000, User: "uploader",
	}}
	if !reflect.DeepEqual(results, want) {
		t.Fatalf("unexpected results:\nwant: %#v\n got: %#v", want, results)
	}
}

const searchPage = `
<html><body>
<table class="table-list"><tbody><tr>
  <td class="coll-1 name"><a href="/torrent/1/Example-Movie-2024-1080p/">Example.Movie.2024.1080p.BluRay.x264.YIFY</a></td>
  <td class="coll-2 seeds">1,234</td>
  <td class="coll-3 leeches">56</td>
  <td class="coll-4 size">1.5 GB</td>
  <td class="coll-5 uploader">uploader</td>
</tr></tbody></table>
</body></html>`
