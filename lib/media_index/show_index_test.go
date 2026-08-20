package index

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func mockShowIndex() *ShowIndex {
	return &ShowIndex{
		shows: map[string]*Show{
			// Game Of Thrones
			"tt0944947": {
				Path: "/home/shows/Game Of Thrones",
				Seasons: map[int]*Season{
					2: {
						Path: "/home/shows/Game Of Thrones/Season 2",
						Episodes: map[int]*Episode{
							2: {
								Path: "/home/shows/Game Of Thrones/Season 2/s02e02.mp4",
								Subtitles: []*Subtitle{
									{
										Size: 1000000,
										Lang: polochon.FR,
									},
									{
										Size: 1000000,
										Lang: polochon.EN,
									},
								},
							},
						},
					},
					1: {
						Path: "/home/shows/Game Of Thrones/Season 1",
						Episodes: map[int]*Episode{
							2: {
								Path: "/home/shows/Game Of Thrones/Season 1/s01e02.mp4",
							},
							1: {
								Path: "/home/shows/Game Of Thrones/Season 1/s01e01.mp4",
							},
						},
					},
				},
			},
			// The Walking Dead
			"tt1520211": {
				Path: "/home/shows/The Walking Dead",
				Seasons: map[int]*Season{
					2: {
						Path: "/home/shows/The Walking Dead/Season 2",
						Episodes: map[int]*Episode{
							1: {
								Path: "/home/shows/The Walking Dead/Season 2/s02e01.mp4",
							},
						},
					},
				},
			},
			// Vickings
			"tt2306299": {
				Path: "/home/shows/Vikings",
				Seasons: map[int]*Season{
					9: {
						Path: "/home/shows/Vikings/Season 9",
						Episodes: map[int]*Episode{
							18: {
								Path: "/home/shows/Vikings/Season 9/s09e18.mp4",
							},
						},
					},
				},
			},
			// Dexter
			"tt0773262": {},
			// Family Guy
			"tt0182576": {
				Path: "/home/shows/Family Guy",
				Seasons: map[int]*Season{
					2: {},
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
		t.Fatal("episode should be empty")
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
		t.Fatal("episode should not be empty")
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
		t.Fatal("season should be empty")
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
		t.Fatal("show should be empty")
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
				ShowImdbID: "tt0944947", // Game Of Thrones
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
				ShowImdbID: "tt0944947", // Game Of Thrones
				Season:     1,
				Episode:    3,
			},
		},
	} {
		idx := mockShowIndex()
		mock.episode.Path = mock.episodePath

		// Add it to the index
		if err := idx.Add(mock.episode); err != nil {
			t.Fatalf("error while adding show in the index: %q", err)
		}

		// Check
		hasEpisode, err := idx.HasEpisode(mock.episode.ShowImdbID, mock.episode.Season, mock.episode.Episode)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}
		if !hasEpisode {
			t.Fatal("the index should have the episode")
		}

		// Ensures the paths are correct
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
	expected := map[string]*Show{}
	idx.Clear()

	if !reflect.DeepEqual(idx.shows, expected) {
		t.Errorf("expected %+v , got %+v", expected, idx)
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

func TestShowIndexAddSubtitle(t *testing.T) {
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
				ShowImdbID: "tt0944947", // Game Of Thrones
				Season:     3,
				Episode:    1,
			},
		},
	} {
		idx := mockShowIndex()
		mock.episode.Path = mock.episodePath

		sub := polochon.NewSubtitleFromVideo(mock.episode, polochon.FR)
		mock.episode.Subtitles = []*polochon.Subtitle{sub}

		// Add episode it to the index
		if err := idx.Add(mock.episode); err != nil {
			t.Fatalf("error while adding show in the index: %q", err)
		}

		// Check
		hasEpisodeSub, err := idx.HasEpisodeSubtitle(mock.episode.ShowImdbID, mock.episode.Season, mock.episode.Episode, sub)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}
		if !hasEpisodeSub {
			t.Fatal("the index should have the episode's subtitle")
		}
	}
}

func TestShowIndexAddNormalizesShowIDAndTitle(t *testing.T) {
	for _, tc := range []struct {
		name    string
		nfoShow *polochon.Show // embedded show metadata supplied with the episode
	}{
		{name: "supplied show metadata", nfoShow: &polochon.Show{ImdbID: "tt-from-nfo"}},
		{name: "nil show fallback", nfoShow: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			idx := NewShowIndex()
			episode := &polochon.ShowEpisode{
				BaseVideo:  polochon.BaseVideo{File: polochon.File{Path: "/home/shows/Show title/Season 1/episode.mp4"}},
				ShowImdbID: "tt-authoritative",
				ShowTitle:  "Fallback title",
				Season:     1,
				Episode:    1,
				Show:       tc.nfoShow,
			}

			if err := idx.Add(episode); err != nil {
				t.Fatalf("add episode: %q", err)
			}

			cached, err := idx.IndexedShow(episode.ShowImdbID)
			if err != nil {
				t.Fatalf("get indexed show: %q", err)
			}
			if cached.Show == nil {
				t.Fatalf("show ID was not normalized: %+v", cached.Show)
			}
			if cached.ImdbID != episode.ShowImdbID {
				t.Fatalf("cached show ID = %q, want %q", cached.ImdbID, episode.ShowImdbID)
			}
			if cached.Title != episode.ShowTitle {
				t.Fatalf("show title = %q, want %q", cached.Title, episode.ShowTitle)
			}
			if tc.nfoShow != nil {
				if _, err := idx.IndexedShow("tt-from-nfo"); err != ErrNotFound {
					t.Fatalf("stale NFO ID should not be indexed, got error %q", err)
				}
			}

			cachedEpisode, err := idx.Episode(episode.ShowImdbID, episode.Season, episode.Episode)
			if err != nil {
				t.Fatalf("get indexed episode: %q", err)
			}
			if cachedEpisode.ShowEpisode == nil {
				t.Fatalf("episode show ID was not normalized: %+v", cachedEpisode.ShowEpisode)
			}
			// The cached episode's embedded show is only non-nil when NFO metadata
			// was supplied with the episode.
			if tc.nfoShow != nil && (cachedEpisode.Show == nil || cachedEpisode.Show.ImdbID != episode.ShowImdbID) {
				t.Fatalf("episode show ID = %+v, want %q", cachedEpisode.Show, episode.ShowImdbID)
			}
		})
	}
}

func TestShowIndexAddCachesEpisodeMetadata(t *testing.T) {
	idx := NewShowIndex()
	episode := &polochon.ShowEpisode{
		BaseVideo: polochon.BaseVideo{
			File: polochon.File{Path: "/home/shows/Show title/Season 1/episode.mp4", Size: 1234},
			VideoMetadata: polochon.VideoMetadata{
				Quality:      polochon.Quality1080p,
				ReleaseGroup: "R1",
				AudioCodec:   "AAC",
				VideoCodec:   "H.264",
				Container:    "mkv",
			},
		},
		ShowImdbID:    "tt12345",
		ShowTitle:     "Show title",
		Season:        1,
		Episode:       1,
		Title:         "Episode title",
		Plot:          "Episode plot",
		Runtime:       42,
		EpisodeImdbID: "tt-episode",
	}
	if err := idx.Add(episode); err != nil {
		t.Fatalf("add episode: %q", err)
	}

	cached, err := idx.Episode(episode.ShowImdbID, episode.Season, episode.Episode)
	if err != nil {
		t.Fatalf("get indexed episode: %q", err)
	}
	if cached.ShowEpisode == nil {
		t.Fatal("episode metadata was not cached")
	}
	if cached.Quality != polochon.Quality1080p || cached.ReleaseGroup != "R1" ||
		cached.AudioCodec != "AAC" || cached.VideoCodec != "H.264" || cached.Container != "mkv" {
		t.Fatalf("episode video metadata was not cached: quality=%q release_group=%q audio=%q video=%q container=%q",
			cached.Quality, cached.ReleaseGroup, cached.AudioCodec, cached.VideoCodec, cached.Container)
	}
	if cached.Title != "Episode title" || cached.Plot != "Episode plot" ||
		cached.Runtime != 42 || cached.EpisodeImdbID != "tt-episode" {
		t.Fatalf("episode NFO metadata was not cached: %+v", cached.ShowEpisode)
	}
	if cached.Path != episode.Path || cached.Filename != "episode.mp4" || cached.Size != 1234 {
		t.Fatalf("episode index file fields wrong: path=%q filename=%q size=%d", cached.Path, cached.Filename, cached.Size)
	}
}
