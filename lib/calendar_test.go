package polochon

import (
	"testing"
	"time"
)

func TestNewShowCalendar(t *testing.T) {
	imdbID := "tt000001"
	sc := NewShowCalendar(imdbID)

	if sc.ImdbID != imdbID {
		t.Fatal("invalid imdb id")
	}
}

func TestShowCalendarEpisodeAvailable(t *testing.T) {
	now := time.Now()
	past := now.Add(-1 * 60 * time.Second)
	future := now.Add(60 * time.Second)

	tt := []struct {
		episode  *ShowCalendarEpisode
		expected bool
		name     string
	}{
		{
			name:     "episode in the future",
			episode:  &ShowCalendarEpisode{AiredDate: &future},
			expected: false,
		},
		{
			name:     "episode with no aired date",
			episode:  &ShowCalendarEpisode{AiredDate: nil},
			expected: true,
		},
		{
			name:     "episode in the past",
			episode:  &ShowCalendarEpisode{AiredDate: &past},
			expected: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.episode.IsAvailable() != tc.expected {
				t.Error("incorrect result")
			}
		})
	}
}

func TestShowCalendarIsOlder(t *testing.T) {
	tt := []struct {
		episode    *ShowCalendarEpisode
		wishedShow *WishedShow
		expected   bool
		name       string
	}{
		{
			name:       "wishlisted S01E10 older than S02E01",
			episode:    &ShowCalendarEpisode{Season: 2, Episode: 1},
			wishedShow: &WishedShow{Season: 1, Episode: 10},
			expected:   false,
		},
		{
			name:       "wishlisted S02E10 older than S01E03",
			episode:    &ShowCalendarEpisode{Season: 1, Episode: 3},
			wishedShow: &WishedShow{Season: 2, Episode: 10},
			expected:   true,
		},
		{
			name:       "wishlisted S02E10 older than S02E09",
			episode:    &ShowCalendarEpisode{Season: 2, Episode: 9},
			wishedShow: &WishedShow{Season: 2, Episode: 10},
			expected:   true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if tc.episode.IsOlder(tc.wishedShow) != tc.expected {
				t.Error("incorrect result")
			}
		})
	}
}
