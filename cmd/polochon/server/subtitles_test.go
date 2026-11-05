package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gorilla/mux"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/modules/mkvinfo"
)

func TestRejectedEmbeddedSubtitleDoesNotUpdateLibrary(t *testing.T) {
	for _, tc := range []struct {
		name string
		id   string
	}{
		{name: "unknown ID", id: "nonexistent"},
		{name: "mismatched language", id: string(polochon.EN)},
		{name: "no embedded subtitle", id: string(polochon.FR)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := newLibraryServer(t)
			indexed, err := srv.library.GetIndexedMovie("tt0000001")
			if err != nil {
				t.Fatal(err)
			}
			// The existing fixture is an MP4, so mkvinfo cannot supply a subtitle.
			movie := polochon.NewMovieFromFile(polochon.MovieConfig{
				Subtitlers: []polochon.Subtitler{&mkvinfo.MKVInfo{}},
			}, polochon.File{Path: indexed.Path})
			movie.ImdbID = indexed.ImdbID
			original := []*polochon.Subtitle{{Lang: polochon.EN, Embedded: true}}
			movie.SetSubtitles(original)
			before, err := json.Marshal(srv.library.MovieIndex())
			if err != nil {
				t.Fatal(err)
			}

			body, err := json.Marshal(polochon.SubtitleEntry{Source: "mkvinfo", ID: tc.id})
			if err != nil {
				t.Fatal(err)
			}
			req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body)), map[string]string{"lang": string(polochon.FR)})
			rr := httptest.NewRecorder()
			srv.downloadSubtitleByEntry(movie, rr, req)
			if rr.Code != http.StatusInternalServerError {
				t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
			}
			if !reflect.DeepEqual(movie.GetSubtitles(), original) {
				t.Fatal("rejected download changed the video's subtitles")
			}
			after, err := json.Marshal(srv.library.MovieIndex())
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("rejected download changed the media index")
			}
		})
	}
}
