package simkl

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestCalendar(t *testing.T) {
	s := metadataModule(t, map[string]string{
		// Calendar lookup must not require artwork or even a series-level date.
		"/tv/84": `{"title":"Show","type":"show","ids":{"simkl":84,"imdb":"tt002"}}`,
		"/tv/episodes/84": `[
			{"type":"episode","season":1,"episode":1,"aired":true,"date":"2000-01-01T23:00:00-05:00"},
			{"type":"episode","season":1,"episode":2,"aired":false,"date":"2999-01-01T09:00:00+09:00"},
			{"type":"episode","season":1,"episode":3,"aired":true,"date":null},
			{"type":"episode","season":1,"episode":4,"aired":false,"date":null},
			{"type":"episode","season":1,"episode":5},
			{"type":"episode","season":1,"episode":1,"aired":true},
			{"type":"special","aired":true},
			{"type":"episode","episode":6,"aired":true}
		]`,
	})
	show := &polochon.Show{ImdbID: "tt002", Title: "Original"}
	before := *show
	calendar, err := s.GetShowCalendar(context.Background(), show)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*show, before) {
		t.Fatal("calendar mutated its input")
	}
	if calendar.ImdbID != "tt002" || len(calendar.Episodes) != 3 {
		t.Fatalf("unexpected calendar: %+v", calendar)
	}
	first, future, undated := calendar.Episodes[0], calendar.Episodes[1], calendar.Episodes[2]
	want, _ := time.Parse(time.RFC3339, "2000-01-02T04:00:00Z")
	if first.Season != 1 || first.Episode != 1 || first.AiredDate == nil || !first.AiredDate.Equal(want) || !first.IsAvailable() {
		t.Errorf("historical calendar entry: %+v", first)
	}
	if future.Episode != 2 || future.AiredDate == nil || future.IsAvailable() {
		t.Errorf("future calendar entry: %+v", future)
	}
	if undated.Episode != 3 || undated.AiredDate != nil || !undated.IsAvailable() {
		t.Errorf("aired, undated calendar entry: %+v", undated)
	}
}

func TestCalendarFailures(t *testing.T) {
	for _, tc := range []struct {
		name      string
		overrides map[string]string
		want      error
	}{
		{"unknown show", map[string]string{"/tv/84": `[]`}, polochon.ErrCalendarNotFound},
		{"malformed date", map[string]string{"/tv/episodes/84": `[{"type":"episode","season":1,"episode":1,"date":"invalid"}]`}, ErrInvalidResponse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := metadataModule(t, tc.overrides)
			if _, err := s.GetShowCalendar(context.Background(), &polochon.Show{ImdbID: "tt002"}); err != tc.want {
				t.Errorf("calendar = %v, want %v", err, tc.want)
			}
		})
	}
	s := metadataModule(t, map[string]string{"/tv/episodes/84": `[]`})
	calendar, err := s.GetShowCalendar(context.Background(), &polochon.Show{ImdbID: "tt002"})
	if err != nil || calendar == nil || len(calendar.Episodes) != 0 {
		t.Errorf("empty calendar = %+v, %v", calendar, err)
	}
	if _, err := s.GetShowCalendar(context.Background(), nil); err != ErrInvalidArgument {
		t.Errorf("nil show = %v", err)
	}
	if _, err := (&Simkl{}).GetShowCalendar(context.Background(), &polochon.Show{}); err != ErrMissingClientID {
		t.Errorf("unconfigured calendar = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.GetShowCalendar(ctx, &polochon.Show{ImdbID: "tt002"}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled calendar = %v", err)
	}
	s = testModule(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "//simkl.com/anime/1/example")
		w.WriteHeader(http.StatusMovedPermanently)
	})
	if _, err := s.GetShowCalendar(context.Background(), &polochon.Show{ImdbID: "tt-anime"}); err != polochon.ErrCalendarNotFound {
		t.Errorf("anime calendar = %v", err)
	}
}
