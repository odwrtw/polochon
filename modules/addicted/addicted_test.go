package addicted

import (
	"errors"
	"io"
	"testing"

	"github.com/odwrtw/addicted"
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

// TestBuildAndParseToken covers the token round-trip.
func TestBuildAndParseToken(t *testing.T) {
	for _, tc := range []struct {
		name     string
		subtitle addicted.Subtitle
		wantLink string
	}{
		{
			name: "basic token",
			subtitle: addicted.Subtitle{
				Title:           "Black Mirror",
				HearingImpaired: false,
				Download:        1234,
				Link:            "/updated/5/1234/5",
			},
			wantLink: "/updated/5/1234/5",
		},
		{
			name: "hearing impaired",
			subtitle: addicted.Subtitle{
				Title:           "The Wire",
				HearingImpaired: true,
				Download:        42,
				Link:            "/updated/3/99/1",
			},
			wantLink: "/updated/3/99/1",
		},
		{
			name: "title containing separator",
			subtitle: addicted.Subtitle{
				Title:           "Gray - A Show",
				HearingImpaired: false,
				Download:        0,
				Link:            "/updated/1/2/3",
			},
			wantLink: "/updated/1/2/3",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := buildToken(tc.subtitle)
			link, err := parseTokenLink(token)
			if err != nil {
				t.Fatalf("parseTokenLink(%q) unexpected error: %v", token, err)
			}
			if link != tc.wantLink {
				t.Fatalf("parseTokenLink() = %q, want %q", link, tc.wantLink)
			}
		})
	}
}

// TestParseTokenLinkInvalid checks that a malformed token returns ErrInvalidToken.
func TestParseTokenLinkInvalid(t *testing.T) {
	for _, tc := range []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"no separator", "BlackMirrorHearingImpaired:false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseTokenLink(tc.token)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("parseTokenLink(%q) err = %v, want ErrInvalidToken", tc.token, err)
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
	entry := &polochon.SubtitleEntry{Token: buildToken(addicted.Subtitle{Link: "/updated/1/2/3"})}
	_, err := a.DownloadSubtitle("not a video", entry, fakeLog)
	if err == nil {
		t.Fatal("DownloadSubtitle() expected error, got nil")
	}
}

// TestDownloadSubtitleInvalidToken checks that a bad token is rejected before
// any network call.
func TestDownloadSubtitleInvalidToken(t *testing.T) {
	a := &addictedProxy{}
	episode := polochon.NewShowEpisodeFromFile(polochon.ShowConfig{}, polochon.File{
		Path: "/shows/Breaking.Bad/S01E01.mkv",
	})
	entry := &polochon.SubtitleEntry{Token: "malformed-token-no-separator"}
	_, err := a.DownloadSubtitle(episode, entry, fakeLog)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("DownloadSubtitle() err = %v, want ErrInvalidToken", err)
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
