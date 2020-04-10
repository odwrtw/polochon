package polochon

import (
	"reflect"
	"testing"
)

func TestTorrentRatio(t *testing.T) {
	tt := []struct {
		name     string
		torrent  *Torrent
		ratio    float32
		expected bool
	}{
		{
			name: "torrent is not finished",
			torrent: &Torrent{
				Status: &TorrentStatus{IsFinished: false},
			},
			expected: false,
		},
		{
			name:     "torrent is has no result",
			torrent:  &Torrent{},
			expected: false,
		},
		{
			name:  "torrent has reached the expected ratio",
			ratio: 2,
			torrent: &Torrent{
				Status: &TorrentStatus{
					IsFinished: true,
					Ratio:      3,
				},
			},
			expected: true,
		},
		{
			name:  "torrent has not yet reached the expected ratio",
			ratio: 1,
			torrent: &Torrent{
				Status: &TorrentStatus{
					IsFinished: true,
					Ratio:      0.3,
				},
			},
			expected: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.torrent.RatioReached(tc.ratio)
			if got != tc.expected {
				t.Fatalf("expected %t, got %t", tc.expected, got)
			}
		})
	}
}

func TestFilterTorrents(t *testing.T) {
	t1 := &Torrent{Quality: Quality720p, Result: &TorrentResult{Seeders: 2}}
	t2 := &Torrent{Quality: Quality1080p, Result: &TorrentResult{Seeders: 200}}
	t3 := &Torrent{Quality: Quality1080p, Result: &TorrentResult{Seeders: 100}}
	t4 := &Torrent{Quality: Quality720p, Result: &TorrentResult{Seeders: 50}}
	t5 := &Torrent{Quality: Quality720p}
	input := []*Torrent{t1, t2, t3, t4, t5}

	expected := []*Torrent{t2, t4}

	got := FilterTorrents(input)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %#v, got %#v", expected, got)
	}
}

func TestChooseTorrentFromQualities(t *testing.T) {
	t1 := &Torrent{Quality: Quality3D}
	t2 := &Torrent{Quality: Quality480p}
	t3 := &Torrent{Quality: Quality720p}
	t4 := &Torrent{Quality: Quality1080p}
	t5 := &Torrent{Quality: Quality1080p}
	input := []*Torrent{t1, t2, t3, t4, t5}

	expected := t4

	got := ChooseTorrentFromQualities(input, []Quality{Quality1080p, Quality720p})
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %#v, got %#v", expected, got)
	}
}

func TestChooseTorrentFromQualitiesNotFound(t *testing.T) {
	input := []*Torrent{
		{Quality: Quality3D},
		{Quality: Quality480p},
	}

	got := ChooseTorrentFromQualities(input, []Quality{Quality1080p, Quality720p})
	if got != nil {
		t.Fatalf("expected no result, got %#v", got)
	}
}

func TestHasVideo(t *testing.T) {
	// Valid movie
	validMovieTorrent := &Torrent{
		ImdbID:  "tt000000",
		Type:    "movie",
		Quality: Quality1080p,
	}

	// Valid episode
	validEpisodeTorrent := &Torrent{
		ImdbID:  "tt000000",
		Type:    "episode",
		Quality: Quality720p,
		Season:  1,
		Episode: 3,
	}

	expectedShow := &Show{ImdbID: validEpisodeTorrent.ImdbID}
	expectedEpisode := &ShowEpisode{
		ShowImdbID: validEpisodeTorrent.ImdbID,
		Season:     validEpisodeTorrent.Season,
		Episode:    validEpisodeTorrent.Episode,
		VideoMetadata: VideoMetadata{
			Quality: validEpisodeTorrent.Quality,
		},
		Torrents: []*Torrent{validEpisodeTorrent},
		Show:     expectedShow,
	}
	expectedShow.Episodes = []*ShowEpisode{expectedEpisode}

	tt := []struct {
		name     string
		torrent  *Torrent
		expected Video
	}{
		{
			name:     "torrent without type",
			torrent:  &Torrent{ImdbID: "tt000000"},
			expected: nil,
		},
		{
			name:     "torrent with invliad type",
			torrent:  &Torrent{ImdbID: "tt000000", Type: "invalid"},
			expected: nil,
		},
		{
			name:    "valid movie",
			torrent: validMovieTorrent,
			expected: &Movie{
				ImdbID: validMovieTorrent.ImdbID,
				VideoMetadata: VideoMetadata{
					Quality: validMovieTorrent.Quality,
				},
				Torrents: []*Torrent{validMovieTorrent},
			},
		},
		{
			name: "episode with missing season",
			torrent: &Torrent{
				ImdbID:  "tt000000",
				Type:    "episode",
				Episode: 2,
			},
			expected: nil,
		},
		{
			name: "episode with missing episode number",
			torrent: &Torrent{
				ImdbID: "tt000000",
				Type:   "episode",
				Season: 1,
			},
			expected: nil,
		},
		{
			name:     "episode with missing episode number",
			torrent:  validEpisodeTorrent,
			expected: expectedEpisode,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.torrent.Video()
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("expected %+v, got %+v", tc.expected, got)
			}
		})
	}
}
