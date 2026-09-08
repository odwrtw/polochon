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

func TestGetTorrentsForMovie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/search/"):
			_, _ = w.Write([]byte(searchPage))
		case strings.HasPrefix(r.URL.Path, "/torrent/1/"):
			_, _ = w.Write([]byte(`<a href="magnet:?xt=urn:btih:abc">Magnet</a>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	x := &X1337{}
	if err := x.InitWithParams(&Params{
		URLs:          []string{server.URL},
		ReleaseGroups: []string{"yify"},
	}); err != nil {
		t.Fatal(err)
	}

	movie := polochon.NewMovie(polochon.MovieConfig{})
	movie.Title = "Example Movie"
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

func TestReleaseGroupFilter(t *testing.T) {
	x := &X1337{}
	if err := x.InitWithParams(&Params{
		URLs:          []string{"https://1337x.example"},
		ReleaseGroups: []string{"NTb", " FraMeSToR "},
	}); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		group string
		want  bool
	}{
		{group: "ntb", want: true},
		{group: "framestor", want: true},
		{group: "YTS", want: false},
		{group: "", want: false},
	} {
		if got := x.acceptsReleaseGroup(test.group); got != test.want {
			t.Errorf("acceptsReleaseGroup(%q) = %t, want %t", test.group, got, test.want)
		}
	}
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
