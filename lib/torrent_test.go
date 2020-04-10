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
	input := []*Torrent{t1, t2, t3, t4}

	expected := []*Torrent{t4, t2}

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
