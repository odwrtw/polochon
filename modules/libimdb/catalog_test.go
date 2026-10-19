package libimdb

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"path/filepath"
	"testing"

	imdb "github.com/odwrtw/libimdb"

	polochon "github.com/odwrtw/polochon/lib"
)

func testCatalogModule(t testing.TB) *Catalog {
	t.Helper()
	data := map[imdb.Dataset]string{
		imdb.Basics: "tconst\ttitleType\tprimaryTitle\toriginalTitle\tisAdult\tstartYear\tendYear\truntimeMinutes\tgenres\n" +
			"tt1\tmovie\tExample\tOriginal Example\t0\t2000\t\\N\t90\tDrama\n" +
			"tt2\tmovie\tExample\tExample\t0\t2020\t\\N\t120\tComedy\n" +
			"tt3\ttvSeries\tExample\tExample\t0\t2001\t\\N\t45\tDrama\n" +
			"tt4\ttvSeries\tExample\tExample\t0\t2021\t\\N\t60\tComedy\n" +
			"tt5\tmovie\tUnknown Movie\t\\N\t\\N\t\\N\t\\N\t\\N\t\\N\n" +
			"tt6\ttvSeries\tUnknown Show\t\\N\t\\N\t\\N\t\\N\t\\N\t\\N\n",
		imdb.Aliases: "titleId\tordering\ttitle\tregion\tlanguage\ttypes\tattributes\tisOriginalTitle\n",
		imdb.Ratings: "tconst\taverageRating\tnumVotes\ntt1\t8.5\t100\ntt3\t7.5\t200\n",
	}
	source := imdb.SourceFunc(func(ctx context.Context, dataset imdb.Dataset) (io.ReadCloser, imdb.SourceMetadata, error) {
		if err := ctx.Err(); err != nil {
			return nil, imdb.SourceMetadata{}, err
		}
		var buffer bytes.Buffer
		writer := gzip.NewWriter(&buffer)
		if _, err := io.WriteString(writer, data[dataset]); err != nil {
			return nil, imdb.SourceMetadata{}, err
		}
		if err := writer.Close(); err != nil {
			return nil, imdb.SourceMetadata{}, err
		}
		return io.NopCloser(bytes.NewReader(buffer.Bytes())), imdb.SourceMetadata{}, nil
	})
	path := filepath.Join(t.TempDir(), "catalog.sqlite")
	if _, err := imdb.Update(context.Background(), path, imdb.UpdateOptions{Source: source}); err != nil {
		t.Fatal(err)
	}
	return testModule(t, path)
}

func TestGetDetailsStartYear(t *testing.T) {
	module := testCatalogModule(t)
	t.Run("movie", func(t *testing.T) {
		movie := &polochon.Movie{Title: "Example", Year: 2020}
		if err := module.GetDetails(context.Background(), movie); err != nil {
			t.Fatal(err)
		}
		if movie.ImdbID != "tt2" || movie.Year != 2020 {
			t.Fatalf("unexpected movie: %+v", movie)
		}
	})
	t.Run("show", func(t *testing.T) {
		show := &polochon.Show{Title: "Example", Year: 2021}
		if err := module.GetDetails(context.Background(), show); err != nil {
			t.Fatal(err)
		}
		if show.ImdbID != "tt4" || show.Year != 2021 {
			t.Fatalf("unexpected show: %+v", show)
		}
	})
	t.Run("episode parent", func(t *testing.T) {
		show := &polochon.Show{Title: "Example", Year: 2001}
		episode := &polochon.ShowEpisode{ShowTitle: "Example", Show: show, Season: 1, Episode: 2}
		if err := module.GetDetails(context.Background(), episode); err != nil {
			t.Fatal(err)
		}
		if episode.ShowImdbID != "tt3" || show.ImdbID != "tt3" || episode.Show != show || show.Year != 2001 {
			t.Fatalf("unexpected episode/parent: %+v / %+v", episode, show)
		}
	})
	t.Run("ambiguous without year", func(t *testing.T) {
		movie := &polochon.Movie{Title: "Example"}
		if err := module.GetDetails(context.Background(), movie); !errors.Is(err, imdb.ErrAmbiguous) {
			t.Fatalf("got %v, want ErrAmbiguous", err)
		}
		if movie.ImdbID != "" {
			t.Fatalf("ambiguous resolution supplied an identity: %+v", movie)
		}
	})
	t.Run("no match for supplied year", func(t *testing.T) {
		movie := &polochon.Movie{Title: "Example", Year: 2010}
		if err := module.GetDetails(context.Background(), movie); !errors.Is(err, imdb.ErrNotFound) {
			t.Fatalf("got %v, want ErrNotFound", err)
		}
		if movie.ImdbID != "" || movie.Year != 2010 {
			t.Fatalf("failed resolution changed movie: %+v", movie)
		}
	})
}

func TestGetDetailsLookupYear(t *testing.T) {
	module := testCatalogModule(t)
	movie := &polochon.Movie{ImdbID: "tt1"}
	if err := module.GetDetails(context.Background(), movie); err != nil {
		t.Fatal(err)
	}
	if movie.Year != 2000 || movie.Title != "Example" || movie.OriginalTitle != "Original Example" {
		t.Fatalf("unexpected movie hints: %+v", movie)
	}
	show := &polochon.Show{ImdbID: "tt3"}
	if err := module.GetDetails(context.Background(), show); err != nil {
		t.Fatal(err)
	}
	if show.Year != 2001 || show.Title != "Example" {
		t.Fatalf("unexpected show hints: %+v", show)
	}
}

func TestSearchCatalogFields(t *testing.T) {
	module := testCatalogModule(t)
	movies, err := module.SearchMovie(context.Background(), "Original Example")
	if err != nil || len(movies) != 1 {
		t.Fatalf("movie search: %v, %v", movies, err)
	}
	movie := movies[0]
	if movie.ImdbID != "tt1" || movie.Title != "Example" || movie.OriginalTitle != "Original Example" ||
		movie.Year != 2000 || movie.Runtime != 90 || movie.Rating != 8.5 || movie.Votes != 100 ||
		len(movie.Genres) != 1 || movie.Genres[0] != "Drama" {
		t.Fatalf("unexpected movie fields: %+v", movie)
	}
	shows, err := module.SearchShow(context.Background(), "Example")
	if err != nil || len(shows) != 2 {
		t.Fatalf("show search: %v, %v", shows, err)
	}
	show := shows[0]
	if show.ImdbID != "tt3" || show.Title != "Example" || show.Year != 2001 || show.Rating != 7.5 {
		t.Fatalf("unexpected show fields: %+v", show)
	}
	movies, err = module.SearchMovie(context.Background(), "Unknown Movie")
	if err != nil || len(movies) != 1 {
		t.Fatalf("unknown movie search: %v, %v", movies, err)
	}
	movie = movies[0]
	if movie.Year != 0 || movie.Runtime != 0 || movie.Rating != 0 || movie.Votes != 0 || movie.OriginalTitle != "" || movie.Genres != nil {
		t.Fatalf("missing movie facts were not preserved: %+v", movie)
	}
	shows, err = module.SearchShow(context.Background(), "Unknown Show")
	if err != nil || len(shows) != 1 {
		t.Fatalf("unknown show search: %v, %v", shows, err)
	}
	if shows[0].Year != 0 || shows[0].Rating != 0 {
		t.Fatalf("missing show facts were not preserved: %+v", shows[0])
	}
}
