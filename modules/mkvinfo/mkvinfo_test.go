package mkvinfo

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestSubtitleRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name  string
		video polochon.Video
	}{
		{name: "movie", video: &polochon.Movie{}},
		{name: "episode", video: &polochon.ShowEpisode{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "video.mkv")
			writeTestMKV(t, path,
				subtitleTrack(1),
				subtitleTrack(2, testElement(languageIETFElement, []byte("fr-FR"))),
			)
			tc.video.SetFile(polochon.File{Path: path})
			m := &MKVInfo{}
			for _, lang := range []polochon.Language{polochon.EN, polochon.FR} {
				entries, err := m.ListSubtitles(t.Context(), tc.video, lang)
				if err != nil {
					t.Fatal(err)
				}
				if len(entries) != 1 || entries[0].ID != string(lang) || !entries[0].Embedded || entries[0].Language != lang {
					t.Fatalf("unexpected entries: %+v", entries)
				}
				for _, entry := range []*polochon.SubtitleEntry{
					entries[0],
					{ID: string(lang), Language: lang}, // Source and descriptive fields are not required.
				} {
					sub, err := m.DownloadSubtitle(t.Context(), tc.video, entry)
					if err != nil {
						t.Fatal(err)
					}
					if sub == nil || !sub.Embedded || sub.Lang != lang || sub.Video != tc.video {
						t.Fatalf("unexpected downloaded subtitle: %+v", sub)
					}
				}
				sub, err := m.GetSubtitle(t.Context(), tc.video, lang)
				if err != nil || sub == nil || !sub.Embedded || sub.Lang != lang || sub.Video != tc.video {
					t.Fatalf("GetSubtitle(%q) = %+v, %v", lang, sub, err)
				}
			}
		})
	}
}

func TestDownloadSubtitleInvalidEntry(t *testing.T) {
	for _, tc := range []struct {
		name  string
		entry *polochon.SubtitleEntry
	}{
		{name: "nil"},
		{name: "empty", entry: &polochon.SubtitleEntry{}},
		{name: "empty ID", entry: &polochon.SubtitleEntry{Language: polochon.FR}},
		{name: "empty language", entry: &polochon.SubtitleEntry{ID: string(polochon.FR)}},
		{name: "unknown ID", entry: &polochon.SubtitleEntry{ID: "nonexistent", Language: polochon.FR}},
		{name: "mismatched language", entry: &polochon.SubtitleEntry{ID: string(polochon.EN), Language: polochon.FR}},
		{name: "unsupported language", entry: &polochon.SubtitleEntry{ID: "de_DE", Language: "de_DE"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sub, err := (&MKVInfo{}).DownloadSubtitle(t.Context(), &polochon.Movie{}, tc.entry)
			if err != ErrInvalidSubtitleEntry || sub != nil {
				t.Fatalf("DownloadSubtitle() = %+v, %v; want nil, ErrInvalidSubtitleEntry", sub, err)
			}
		})
	}
}

func TestSubtitleErrors(t *testing.T) {
	dir := t.TempDir()
	withoutSubtitle := filepath.Join(dir, "without-subtitle.mkv")
	writeTestMKV(t, withoutSubtitle)
	forced := filepath.Join(dir, "forced.mkv")
	writeTestMKV(t, forced, subtitleTrack(1, testElement(flagForcedElement, []byte{1})))
	forcedName := filepath.Join(dir, "forced-name.mkv")
	writeTestMKV(t, forcedName, subtitleTrack(1, testElement(nameElement, []byte("English Forced"))))
	french := filepath.Join(dir, "french.mkv")
	writeTestMKV(t, french, subtitleTrack(1, testElement(languageElement, []byte("fre"))))
	broken := filepath.Join(dir, "broken.mkv")
	if err := os.WriteFile(broken, []byte{0x18, 0x53, 0x80, 0x67}, 0o600); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name  string
		video any
		want  error
	}{
		{name: "not a video", video: "invalid", want: ErrNotAVideo},
		{name: "empty path", video: &polochon.Movie{}, want: ErrMissingFilePath},
		{name: "missing file", video: polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{Path: filepath.Join(dir, "missing.mkv")}), want: os.ErrNotExist},
		{name: "non MKV", video: polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{Path: "video.mp4"}), want: polochon.ErrNotAvailable},
		{name: "no subtitles", video: polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{Path: withoutSubtitle}), want: polochon.ErrNoSubtitleFound},
		{name: "forced flag only", video: polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{Path: forced}), want: polochon.ErrNoSubtitleFound},
		{name: "forced name only", video: polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{Path: forcedName}), want: polochon.ErrNoSubtitleFound},
		{name: "wrong language", video: polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{Path: french}), want: polochon.ErrNoSubtitleFound},
		{name: "broken file", video: polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{Path: broken})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &MKVInfo{}
			entries, listErr := m.ListSubtitles(t.Context(), tc.video, polochon.EN)
			sub, getErr := m.GetSubtitle(t.Context(), tc.video, polochon.EN)
			downloaded, downloadErr := m.DownloadSubtitle(t.Context(), tc.video, &polochon.SubtitleEntry{ID: string(polochon.EN), Language: polochon.EN})
			if entries != nil || sub != nil || downloaded != nil {
				t.Fatal("failed subtitle lookup returned a result")
			}
			for _, err := range []error{listErr, getErr, downloadErr} {
				if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) {
					t.Errorf("got error %v, want %v", err, tc.want)
				}
			}
		})
	}
}

func TestDownloadSubtitleStaleEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "video.mkv")
	writeTestMKV(t, path, subtitleTrack(1))
	video := polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{Path: path})
	m := &MKVInfo{}
	entries, err := m.ListSubtitles(t.Context(), video, polochon.EN)
	if err != nil {
		t.Fatal(err)
	}
	writeTestMKV(t, path)
	sub, err := m.DownloadSubtitle(t.Context(), video, entries[0])
	if err != polochon.ErrNoSubtitleFound || sub != nil {
		t.Fatalf("DownloadSubtitle() = %+v, %v; want nil, ErrNoSubtitleFound", sub, err)
	}
}
