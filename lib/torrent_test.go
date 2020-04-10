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
				IsFinished: false,
			},
			expected: false,
		},
		{
			name:  "torrent has reached the expected ratio",
			ratio: 2,
			torrent: &Torrent{
				IsFinished: true,
				Ratio:      3,
			},
			expected: true,
		},
		{
			name:  "torrent has not yet reached the expected ratio",
			ratio: 1,
			torrent: &Torrent{
				IsFinished: true,
				Ratio:      0.3,
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
	t1 := &Torrent{Quality: Quality720p, Seeders: 2}
	t2 := &Torrent{Quality: Quality1080p, Seeders: 200}
	t3 := &Torrent{Quality: Quality1080p, Seeders: 100}
	t4 := &Torrent{Quality: Quality720p, Seeders: 50}
	input := []*Torrent{t1, t2, t3, t4}

	expected := []*Torrent{t4, t2}

	got := FilterTorrents(input)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %#v, got %#v", expected, got)
	}
}
