package transmission

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestLabels(t *testing.T) {
	tt := []struct {
		name     string
		metadata *polochon.DownloadableMetadata
		expected []string
	}{
		{
			name:     "no metadata",
			metadata: nil,
			expected: nil,
		},
		{
			name:     "no imdb id",
			metadata: &polochon.DownloadableMetadata{Type: "movie"},
			expected: nil,
		},
		{
			name: "invalid type",
			metadata: &polochon.DownloadableMetadata{
				ImdbID:  "tt000000",
				Quality: polochon.Quality720p,
				Type:    "test",
			},
			expected: nil,
		},
		{
			name: "valid movie",
			metadata: &polochon.DownloadableMetadata{
				ImdbID:  "tt000000",
				Quality: polochon.Quality720p,
				Type:    "movie",
			},
			expected: []string{
				"type=movie",
				"imdb_id=tt000000",
				"quality=720p",
			},
		},
		{
			name: "invalid episode",
			metadata: &polochon.DownloadableMetadata{
				ImdbID:  "tt000000",
				Quality: polochon.Quality720p,
				Type:    "episode",
			},
			expected: nil,
		},
		{
			name: "valid episode",
			metadata: &polochon.DownloadableMetadata{
				ImdbID:  "tt000000",
				Quality: polochon.Quality720p,
				Type:    "episode",
				Season:  1,
				Episode: 3,
			},
			expected: []string{
				"type=episode",
				"imdb_id=tt000000",
				"quality=720p",
				"season=1",
				"episode=3",
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := labels(tc.metadata)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("expected %+v, got %+v", tc.expected, got)
			}
		})
	}
}

func TestMetadata(t *testing.T) {
	tt := []struct {
		name     string
		labels   []string
		expected *polochon.DownloadableMetadata
	}{
		{
			name:     "no labels",
			labels:   nil,
			expected: nil,
		},
		{
			name: "missing movie quality",
			labels: []string{
				"type=movie",
				"imdb_id=tt000000",
			},
			expected: nil,
		},
		{
			name: "valid movie",
			labels: []string{
				"type=movie",
				"imdb_id=tt000000",
				"quality=720p",
			},
			expected: &polochon.DownloadableMetadata{
				ImdbID:  "tt000000",
				Quality: polochon.Quality720p,
				Type:    "movie",
			},
		},
		{
			name: "valid episode",
			labels: []string{
				"type=episode",
				"imdb_id=tt000000",
				"quality=720p",
				"season=1",
				"episode=3",
			},
			expected: &polochon.DownloadableMetadata{
				ImdbID:  "tt000000",
				Quality: polochon.Quality720p,
				Type:    "episode",
				Season:  1,
				Episode: 3,
			},
		},
		{
			name: "invalid episode season",
			labels: []string{
				"type=episode",
				"imdb_id=tt000000",
				"quality=720p",
				"season=invalid",
				"episode=3",
			},
			expected: nil,
		},
		{
			name: "invalid episode number",
			labels: []string{
				"type=episode",
				"imdb_id=tt000000",
				"quality=720p",
				"season=1",
				"episode=invalid",
			},
			expected: nil,
		},
		{
			name: "invalid quality",
			labels: []string{
				"type=episode",
				"imdb_id=tt000000",
				"quality=UltraHD++",
				"season=1",
				"episode=3",
			},
			expected: nil,
		},
		{
			name: "invalid labels",
			labels: []string{
				"invalid",
			},
			expected: nil,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := metadata(tc.labels)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("expected %+v, got %+v", tc.expected, got)
			}
		})
	}
}
