package index

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func mockShowIndex() *ShowIndex {
	return &ShowIndex{
		shows: map[string]*polochon.Show{
			// Game Of Thrones
			"tt0944947": {
				ImdbID: "tt0944947",
				Seasons: map[int]*polochon.ShowSeason{
					2: {
						Season:     2,
						ShowImdbID: "tt0944947",
						Episodes: map[int]*polochon.ShowEpisode{
							2: {
								BaseVideo: polochon.BaseVideo{
									File: polochon.File{Path: "/home/shows/Game Of Thrones/Season 2/s02e02.mp4"},
									Subtitles: []*polochon.Subtitle{
										{File: polochon.File{Size: 1000000}, Lang: polochon.FR},
										{File: polochon.File{Size: 1000000}, Lang: polochon.EN},
									},
								},
								Season:  2,
								Episode: 2,
							},
						},
					},
					1: {
						Season:     1,
						ShowImdbID: "tt0944947",
						Episodes: map[int]*polochon.ShowEpisode{
							2: {
								BaseVideo: polochon.BaseVideo{File: polochon.File{Path: "/home/shows/Game Of Thrones/Season 1/s01e02.mp4"}},
								Season:    1,
								Episode:   2,
							},
							1: {
								BaseVideo: polochon.BaseVideo{File: polochon.File{Path: "/home/shows/Game Of Thrones/Season 1/s01e01.mp4"}},
								Season:    1,
								Episode:   1,
							},
						},
					},
				},
			},
			// The Walking Dead
			"tt1520211": {
				ImdbID: "tt1520211",
				Seasons: map[int]*polochon.ShowSeason{
					2: {
						Season:     2,
						ShowImdbID: "tt1520211",
						Episodes: map[int]*polochon.ShowEpisode{
							1: {
								BaseVideo: polochon.BaseVideo{File: polochon.File{Path: "/home/shows/The Walking Dead/Season 2/s02e01.mp4"}},
								Season:    2,
								Episode:   1,
							},
						},
					},
				},
			},
			// Vikings
			"tt2306299": {
				ImdbID: "tt2306299",
				Seasons: map[int]*polochon.ShowSeason{
					9: {
						Season:     9,
						ShowImdbID: "tt2306299",
						Episodes: map[int]*polochon.ShowEpisode{
							18: {
								BaseVideo: polochon.BaseVideo{File: polochon.File{Path: "/home/shows/Vikings/Season 9/s09e18.mp4"}},
								Season:    9,
								Episode:   18,
							},
						},
					},
				},
			},
			// Dexter — no seasons (empty show)
			"tt0773262": {
				ImdbID:  "tt0773262",
				Seasons: map[int]*polochon.ShowSeason{},
			},
			// Family Guy — season 2 exists but empty (no episodes)
			"tt0182576": {
				ImdbID: "tt0182576",
				Seasons: map[int]*polochon.ShowSeason{
					2: {
						Season:     2,
						ShowImdbID: "tt0182576",
						Episodes:   map[int]*polochon.ShowEpisode{},
					},
				},
			},
		},
	}
}

func TestShowIndexHasShow(t *testing.T) {
	idx := mockShowIndex()

	for _, mock := range []struct {
		imdbID   string
		expected bool
	}{
		{"tt0944947", true},
		{"not_in_index", false},
	} {
		got, err := idx.HasShow(mock.imdbID)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}

		if mock.expected != got {
			t.Errorf("expected %t, got %t for %s", mock.expected, got, mock.imdbID)
		}
	}
}

func TestShowIndexHasSeason(t *testing.T) {
	idx := mockShowIndex()

	for _, mock := range []struct {
		imdbID   string
		season   int
		expected bool
	}{
		{"tt0944947", 1, true},
		{"tt0944947", 2, true},
		{"tt0182576", 2, true},
		{"tt0182576", 3, false},
		{"tt912918291", 1, false},
	} {
		got, err := idx.HasSeason(mock.imdbID, mock.season)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}

		if mock.expected != got {
			t.Errorf("expected %t, got %t for %s season %d", mock.expected, got, mock.imdbID, mock.season)
		}
	}
}

func TestShowIndexHasEpisode(t *testing.T) {
	idx := mockShowIndex()

	for _, mock := range []struct {
		imdbID   string
		season   int
		episode  int
		expected bool
	}{
		{"tt0944947", 1, 1, true},
		{"tt0944947", 1, 2, true},
		{"tt0944947", 1, 3, false},
		{"tt1520211", 1, 3, false},
		{"tt1520211", 2, 1, true},
		{"tt11111", 2, 1, false},
	} {
		got, err := idx.HasEpisode(mock.imdbID, mock.season, mock.episode)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}

		if mock.expected != got {
			t.Errorf("expected %t, got %t for %s s%d e%d", mock.expected, got, mock.imdbID, mock.season, mock.episode)
		}
	}
}

func TestShowIndexIsShowEmpty(t *testing.T) {
	idx := mockShowIndex()
	for id, expected := range map[string]bool{
		"tt456789":  true,
		"tt0773262": true,
		"tt0944947": false,
		"tt1520211": false,
	} {
		got, err := idx.IsShowEmpty(id)
		if err != nil {
			t.Fatalf("failed to check if the show is empty: %q", err)
		}
		if expected != got {
			t.Errorf("expected %t, got %t for %s", expected, got, id)
		}
	}
}

func TestShowIndexIsSeasonEmpty(t *testing.T) {
	idx := mockShowIndex()
	for _, mock := range []struct {
		imdbID   string
		season   int
		expected bool
	}{
		{"tt1520211", 2, false},
		{"tt1520211", 3, true},
		{"tt0182576", 2, true},
		{"tt0182576", 1, true},
		{"tt0944947", 1, false},
		{"tt0944947", 2, false},
	} {
		got, err := idx.IsSeasonEmpty(mock.imdbID, mock.season)
		if err != nil {
			t.Fatalf("failed to check if the show season is empty: %q", err)
		}
		if mock.expected != got {
			t.Errorf("expected %t, got %t for %s and %d", mock.expected, got, mock.imdbID, mock.season)
		}
	}
}

func TestShowIndexRemoveEpisode(t *testing.T) {
	idx := mockShowIndex()
	id := "tt0944947"
	season := 1
	episode := 2

	inIndex, err := idx.HasEpisode(id, season, episode)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if !inIndex {
		t.Fatal("episode should be in index")
	}

	e := &polochon.ShowEpisode{
		ShowImdbID: id,
		Season:     season,
		Episode:    episode,
	}
	if err := idx.RemoveEpisode(e); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	inIndex, err = idx.HasEpisode(id, season, episode)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if inIndex {
		t.Fatal("episode should not be in index after removal")
	}
}

func TestShowIndexRemoveSeason(t *testing.T) {
	idx := mockShowIndex()

	id := "tt2306299"
	season := 9

	empty, err := idx.IsSeasonEmpty(id, season)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if empty {
		t.Fatal("season should not be empty")
	}

	s := &polochon.Show{ImdbID: id}
	if err := idx.RemoveSeason(s, season); err != nil {
		t.Fatalf("error while removing season from the index: %q", err)
	}

	empty, err = idx.IsSeasonEmpty(id, season)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if !empty {
		t.Fatal("season should be empty after removal")
	}
}

func TestShowIndexRemoveShow(t *testing.T) {
	idx := mockShowIndex()
	id := "tt2306299"

	empty, err := idx.IsShowEmpty(id)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if empty {
		t.Fatal("show should not be empty")
	}

	s := &polochon.Show{ImdbID: id}
	if err := idx.RemoveShow(s); err != nil {
		t.Fatalf("error while removing show from the index: %q", err)
	}

	empty, err = idx.IsShowEmpty(id)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if !empty {
		t.Fatal("show should be empty after removal")
	}
}

func TestShowIndexAdd(t *testing.T) {
	for _, mock := range []struct {
		expectedShowPath   string
		expectedSeasonPath string
		episodePath        string
		episode            *polochon.ShowEpisode
	}{
		{
			// New show, nothing is in the index yet
			expectedShowPath:   "/home/shows/How I Met Your Mother",
			expectedSeasonPath: "/home/shows/How I Met Your Mother/Season 1",
			episodePath:        "/home/shows/How I Met Your Mother/Season 1/s01e01.mp4",
			episode: &polochon.ShowEpisode{
				ShowImdbID: "tt0460649",
				Season:     1,
				Episode:    1,
			},
		},
		{
			// New season, the show is already in the index
			expectedShowPath:   "/home/shows/Game Of Thrones",
			expectedSeasonPath: "/home/shows/Game Of Thrones/Season 3",
			episodePath:        "/home/shows/Game Of Thrones/Season 3/s03e01.mp4",
			episode: &polochon.ShowEpisode{
				ShowImdbID: "tt0944947",
				Season:     3,
				Episode:    1,
			},
		},
		{
			// New episode, the show and the season are already in the index
			expectedShowPath:   "/home/shows/Game Of Thrones",
			expectedSeasonPath: "/home/shows/Game Of Thrones/Season 1",
			episodePath:        "/home/shows/Game Of Thrones/Season 1/s01e03.mp4",
			episode: &polochon.ShowEpisode{
				ShowImdbID: "tt0944947",
				Season:     1,
				Episode:    3,
			},
		},
	} {
		idx := mockShowIndex()
		mock.episode.Path = mock.episodePath

		if err := idx.Add(mock.episode); err != nil {
			t.Fatalf("error while adding show in the index: %q", err)
		}

		hasEpisode, err := idx.HasEpisode(mock.episode.ShowImdbID, mock.episode.Season, mock.episode.Episode)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}
		if !hasEpisode {
			t.Fatal("the index should have the episode")
		}

		showPath, err := idx.ShowPath(mock.episode.ShowImdbID)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}
		if showPath != mock.expectedShowPath {
			t.Errorf("expected show path to be %q, got %q", mock.expectedShowPath, showPath)
		}

		seasonPath, err := idx.SeasonPath(mock.episode.ShowImdbID, mock.episode.Season)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}
		if seasonPath != mock.expectedSeasonPath {
			t.Errorf("expected season path to be %q, got %q", mock.expectedSeasonPath, seasonPath)
		}
	}
}

func TestEmptyShowIndex(t *testing.T) {
	idx := NewShowIndex()
	idx.Clear()

	if len(idx.shows) != 0 {
		t.Errorf("expected empty index after clear, got %d entries", len(idx.shows))
	}
}

func TestShowIDs(t *testing.T) {
	idx := NewShowIndex()
	expected := idx.shows

	got := idx.Index()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %+v , got %+v", expected, got)
	}
}

func TestSeasonList(t *testing.T) {
	idx := mockShowIndex()

	indexedShow, err := idx.IndexedShow("tt0944947")
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	expected := []int{1, 2}
	got := indexedShow.SeasonList()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %+v , got %+v", expected, got)
	}
}

func TestShowIndexHasEpisodeSubtitle(t *testing.T) {
	idx := mockShowIndex()

	for _, mock := range []struct {
		imdbID      string
		season      int
		episode     int
		lang        polochon.Language
		expected    bool
		expectedErr error
	}{
		{"tt0944947", 2, 2, polochon.EN, true, nil},
		{"tt0944947", 2, 2, polochon.FR, true, nil},
		{"tt0944947", 1, 3, polochon.FR, false, ErrNotFound},
		{"tt1520211", 1, 3, polochon.FR, false, ErrNotFound},
		{"tt1520211", 2, 1, polochon.FR, false, nil},
		{"tt11111", 2, 1, polochon.FR, false, ErrNotFound},
	} {
		sub := &polochon.Subtitle{Lang: mock.lang}
		got, err := idx.HasEpisodeSubtitle(mock.imdbID, mock.season, mock.episode, sub)
		if err != mock.expectedErr {
			t.Fatalf("expected error %q, got %q", mock.expectedErr, err)
		}

		if mock.expected != got {
			t.Errorf("expected %t, got %t for %s s%d e%d", mock.expected, got, mock.imdbID, mock.season, mock.episode)
		}
	}
}
