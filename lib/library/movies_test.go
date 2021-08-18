package library

import (
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
)

func TestMovieDir(t *testing.T) {
	library := New(&configuration.Config{})

	tt := []struct {
		name     string
		movie    *polochon.Movie
		expected string
	}{
		{
			name:     "movie without year",
			movie:    &polochon.Movie{Title: "Test"},
			expected: "Test",
		},
		{
			name:     "movie with year",
			movie:    &polochon.Movie{Title: "Test", Year: 2021},
			expected: "Test (2021)",
		},
		{
			name:     "movie with / in the name",
			movie:    &polochon.Movie{Title: "Fahrenheit 9/11"},
			expected: "Fahrenheit 9-11",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := library.getMovieDir(tc.movie)
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
