package tvdb

import (
	"encoding/xml"
	"fmt"
	"reflect"
	"testing"

	"github.com/garfunkel/go-tvdb"
	"github.com/odwrtw/polochon/lib"
)

var testDetailer = &TvDB{}

func TestTvdbUpdateShow(t *testing.T) {
	getShowFromTvdb = func(s *tvdb.Series) error { return nil }
	s := polochon.NewShow(polochon.ShowConfig{})

	// Tvdb input
	input := &tvdb.Series{
		ID:            0x11db5,
		AirsDayOfWeek: "Monday",
		AirsTime:      "9:00 PM",
		ContentRating: "TV-14",
		FirstAired:    "2005-02-06",
		ImdbID:        "tt0397306",
		Language:      "en",
		Network:       "TBS",
		NetworkID:     "",
		Overview:      "Awesome plot",
		Rating:        "8.5",
		RatingCount:   "274",
		Runtime:       "30",
		SeriesID:      "21935",
		SeriesName:    "American Dad!",
		Status:        "Continuing",
		Banner:        "graphical/73141-g10.jpg",
		Fanart:        "fanart/original/73141-12.jpg",
		LastUpdated:   "1430220717",
		Poster:        "posters/73141-8.jpg",
		Zap2ItID:      "EP00716445",
		Seasons: map[uint64][]*tvdb.Episode{
			0x1: {
				{
					ID: 0x5c71d,
					CombinedEpisodeNumber: "8.0",
					CombinedSeason:        0x4,
					DvdEpisodeNumber:      "8.0",
					DvdSeason:             "4",
					EpImgFlag:             "1",
					EpisodeName:           "1600 Candles",
					EpisodeNumber:         0x1,
					FirstAired:            "2008-09-28",
					GuestStars:            "Elizabeth Banks",
					ImdbID:                "",
					Language:              "en",
					Overview:              "Awesome plot",
					ProductionCode:        "3AJN20",
					Rating:                "7.5",
					RatingCount:           "46",
					SeasonNumber:          0x5,
					Filename:              "episodes/73141/378653.jpg",
					LastUpdated:           "1321349913",
					SeasonID:              0x9f99,
					SeriesID:              0x11db5,
				},
			},
		},
	}

	// Expected show values
	expected := polochon.NewShow(polochon.ShowConfig{})
	expected.Title = "American Dad!"
	expected.ShowTitle = "American Dad!"
	expected.Rating = 8.5
	expected.Plot = "Awesome plot"
	expected.URL = fmt.Sprintf("%s/%s/series/%d/all/en.zip", APIendpoint, Token, 73141)
	expected.TvdbID = 73141
	expected.ImdbID = "tt0397306"
	expected.Year = 2005
	expected.Banner = "http://thetvdb.com/banners/graphical/73141-g10.jpg"
	expected.Fanart = "http://thetvdb.com/banners/fanart/original/73141-12.jpg"
	expected.Poster = "http://thetvdb.com/banners/posters/73141-8.jpg"
	expected.Episodes = []*polochon.ShowEpisode{
		{
			XMLName:       xml.Name{Space: "", Local: "episodedetails"},
			Title:         "1600 Candles",
			ShowTitle:     "American Dad!",
			Season:        5,
			Episode:       1,
			TvdbID:        378653,
			Aired:         "2008-09-28",
			Plot:          "Awesome plot",
			Runtime:       30,
			Thumb:         "http://thetvdb.com/banners/episodes/73141/378653.jpg",
			Rating:        7.5,
			ShowImdbID:    "tt0397306",
			ShowTvdbID:    73141,
			EpisodeImdbID: "",
		},
	}

	err := updateShow(s, input)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if !reflect.DeepEqual(s, expected) {
		t.Errorf("Expected: %+v\nGot: %+v", expected, s)
	}
}

func TestTvdbGetShowByName(t *testing.T) {
	s := polochon.NewShow(polochon.ShowConfig{})
	s.Title = "American Dad"

	tvdbGetSeries = func(name string) (seriesList tvdb.SeriesList, err error) {
		return tvdb.SeriesList{
			Series: []*tvdb.Series{
				{SeriesName: "Kill the americans", ImdbID: "tt0832306"},
				{SeriesName: "American Dad!", ImdbID: "tt0397306"},
				{SeriesName: "American Family", ImdbID: "tt9832306"},
			},
		}, nil
	}

	err := getShowByName(s)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if s.ImdbID != "tt0397306" {
		t.Errorf("Expected \"tt0397306\" got %q", s.ImdbID)
	}
}

func TestTvdbGetShowByImdbID(t *testing.T) {
	s := polochon.NewShow(polochon.ShowConfig{})

	tvdbGetShowByImdbID = func(id string) (series *tvdb.Series, err error) {
		return &tvdb.Series{SeriesName: "American Dad!", ImdbID: "tt0397306"}, nil
	}

	err := getShowByImdbID(s)
	if err != ErrMissingShowImdbID {
		t.Fatalf("Expected %q, got %q", ErrMissingShowImdbID, err)
	}

	s.ImdbID = "tt0397306"
	err = getShowByImdbID(s)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if s.Title != "American Dad!" {
		t.Errorf("Expected \"American Dad!\" got %q", s.Title)
	}
}

func TesTvdbGetShowDetails(t *testing.T) {
	s := polochon.NewShow(polochon.ShowConfig{})

	// No arguments
	err := getShowDetails(s)
	if err != ErrNotEnoughArguments {
		t.Fatalf("Expected %q, got %q", ErrNotEnoughArguments, err)
	}

	// Get by title
	s.Title = "American Dad!"
	tvdbGetSeries = func(name string) (seriesList tvdb.SeriesList, err error) {
		return tvdb.SeriesList{
			Series: []*tvdb.Series{{SeriesName: "American Dad!", ImdbID: "tt0397306"}},
		}, nil
	}

	err = getShowDetails(s)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if s.ImdbID != "tt0397306" {
		t.Errorf("Failed to update show by title")
	}

	// Get by imdb id
	s.Title = ""
	tvdbGetShowByImdbID = func(id string) (series *tvdb.Series, err error) {
		return &tvdb.Series{SeriesName: "American Dad!", ImdbID: "tt0397306"}, nil
	}
	getShowFromTvdb = func(s *tvdb.Series) error { return nil }

	err = getShowDetails(s)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if s.Title != "American Dad!" {
		t.Errorf("Failed to update show by imdb id")
	}
}

func TestTvdbGetDetailsFromShow(t *testing.T) {
	s := polochon.NewShow(polochon.ShowConfig{})
	s.ImdbID = "tt0397306"

	tvdbGetShowByImdbID = func(id string) (series *tvdb.Series, err error) {
		return &tvdb.Series{SeriesName: "American Dad!", ImdbID: "tt0397306"}, nil
	}
	getShowFromTvdb = func(s *tvdb.Series) error { return nil }

	// No arguments
	err := testDetailer.GetDetails(s)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}
}

func TestTvdbGetDetailsInvalidArgument(t *testing.T) {
	m := "invalid type"

	// No arguments
	err := testDetailer.GetDetails(m)
	if err != ErrInvalidArgument {
		t.Fatalf("Expected %q got %q", ErrInvalidArgument, err)
	}
}

func TestTvdbGetShowEpisodeDetailsInvalidArgumens(t *testing.T) {
	s := polochon.NewShowEpisode(polochon.ShowConfig{})

	err := getShowEpisodeDetails(s)
	if err != ErrMissingShowEpisodeInformations {
		t.Fatalf("Expected %q got %q", ErrMissingShowEpisodeInformations, err)
	}

	s.Episode = 1
	s.Season = 1
	err = getShowEpisodeDetails(s)
	if err != ErrMissingShowEpisodeInformations {
		t.Fatalf("Expected %q got %q", ErrMissingShowEpisodeInformations, err)
	}
}

func TestTvdbGetShowEpisodeDetailsFailed(t *testing.T) {
	s := polochon.NewShowEpisode(polochon.ShowConfig{})
	s.ShowImdbID = "tt0397306"
	s.Episode = 1
	s.Season = 1

	tvdbGetShowByImdbID = func(id string) (series *tvdb.Series, err error) {
		return &tvdb.Series{SeriesName: "American Dad!", ImdbID: "tt0397306"}, nil
	}
	getShowFromTvdb = func(s *tvdb.Series) error { return nil }

	err := getShowEpisodeDetails(s)
	if err != ErrFailedToUpdateEpisode {
		t.Fatalf("Expected %q got %q", ErrFailedToUpdateEpisode, err)
	}
}

func TestTvdbGetShowEpisodeDetails(t *testing.T) {
	s := polochon.NewShowEpisode(polochon.ShowConfig{})
	s.ShowImdbID = "tt0397306"
	s.Episode = 1
	s.Season = 1

	tvdbGetShowByImdbID = func(id string) (series *tvdb.Series, err error) {
		return &tvdb.Series{
			SeriesName: "American Dad!",
			ImdbID:     "tt0397306",
			Seasons: map[uint64][]*tvdb.Episode{
				0x1: {
					{
						ID:            0x5c71d,
						EpisodeName:   "1600 Candles",
						EpisodeNumber: 0x5,
						FirstAired:    "2008-09-28",
						SeasonNumber:  0x1,
						SeasonID:      0x9f99,
						SeriesID:      0x11db5,
					},
					{
						ID:            0x5c71d,
						EpisodeName:   "Awesome title",
						EpisodeNumber: 0x1,
						FirstAired:    "2008-09-28",
						SeasonNumber:  0x1,
						SeasonID:      0x9f99,
						SeriesID:      0x11db5,
					},
				},
			},
		}, nil
	}
	getShowFromTvdb = func(s *tvdb.Series) error { return nil }

	err := getShowEpisodeDetails(s)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	if s.Title != "Awesome title" {
		t.Fatalf("Failed to get show details expected %q, got %q", "Awesome title", s.Title)
	}
}
