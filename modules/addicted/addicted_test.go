package addicted

import (
	"errors"
	"io"
	"testing"

	"github.com/sirupsen/logrus"

	polochon "github.com/odwrtw/polochon/lib"
)

var fakeLog = logrus.NewEntry(&logrus.Logger{Out: io.Discard})

// TestInitWithParams covers credential validation.
func TestInitWithParams(t *testing.T) {
	for _, tc := range []struct {
		name    string
		params  Params
		wantErr error
	}{
		{
			name:    "missing user and password",
			params:  Params{},
			wantErr: ErrMissingCredentials,
		},
		{
			name:    "missing password",
			params:  Params{User: "greg"},
			wantErr: ErrMissingCredentials,
		},
		{
			name:    "missing user",
			params:  Params{Password: "secret"},
			wantErr: ErrMissingCredentials,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &addictedProxy{}
			err := a.InitWithParams(&tc.params)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("InitWithParams() err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

// TestListSubtitlesWrongType checks that non-ShowEpisode input is rejected.
func TestListSubtitlesWrongType(t *testing.T) {
	a := &addictedProxy{}
	_, err := a.ListSubtitles("not an episode", polochon.EN, fakeLog)
	if !errors.Is(err, polochon.ErrNotAvailable) {
		t.Fatalf("ListSubtitles() err = %v, want ErrNotAvailable", err)
	}
}

// TestDownloadSubtitleWrongType checks that a non-Video input is rejected.
func TestDownloadSubtitleWrongType(t *testing.T) {
	a := &addictedProxy{}
	entry := &polochon.SubtitleEntry{ID: "/updated/1/2/3"}
	_, err := a.DownloadSubtitle("not a video", entry, fakeLog)
	if err == nil {
		t.Fatal("DownloadSubtitle() expected error, got nil")
	}
}

// TestGetSubtitleWrongType checks that non-ShowEpisode input is rejected.
func TestGetSubtitleWrongType(t *testing.T) {
	a := &addictedProxy{}
	_, err := a.GetSubtitle("not an episode", polochon.EN, fakeLog)
	if err == nil {
		t.Fatal("GetSubtitle() expected error, got nil")
	}
}
