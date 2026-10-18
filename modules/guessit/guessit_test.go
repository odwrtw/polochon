package guessit

import (
	"path/filepath"
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestGuess(t *testing.T) {
	movieConf := polochon.MovieConfig{Torrenters: []polochon.Torrenter{nil}}
	showConf := polochon.ShowConfig{Detailers: []polochon.Detailer{nil}}
	for _, tc := range []struct {
		name     string
		title    string
		year     int
		episode  int
		season   int
		metadata polochon.VideoMetadata
	}{
		{
			name: "Big.Buck.Bunny.2008.1080p.BluRay.x264-YIFY.mkv", title: "Big Buck Bunny", year: 2008,
			metadata: polochon.VideoMetadata{Quality: polochon.Quality1080p, VideoCodec: "H.264", ReleaseGroup: "YIFY", Container: "mkv"},
		},
		{
			name: "étoile.de.Mer.2024.FRE.ENG.CHS.CHT.2160p.x265.AAC-GROUP.mkv", title: "étoile de Mer", year: 2024,
			metadata: polochon.VideoMetadata{Quality: polochon.Quality2160p, VideoCodec: "H.265", AudioCodec: "AAC", ReleaseGroup: "GROUP", Container: "mkv"},
		},
		{
			name: "The.Show.2020.S00E03.720p.WEB-DL.H.264.AAC-GROUP.mkv", title: "The Show", year: 2020, episode: 3,
			metadata: polochon.VideoMetadata{Quality: polochon.Quality720p, VideoCodec: "H.264", AudioCodec: "AAC", ReleaseGroup: "GROUP", Container: "mkv"},
		},
		{
			name: "[Team] Example Show - 12 [720p] [A1B2C3D4].mkv", title: "Example Show", episode: 12,
			metadata: polochon.VideoMetadata{Quality: polochon.Quality720p, ReleaseGroup: "Team", Container: "mkv"},
		},
		{
			name: "Example.Show.S02E03.1080p.x265-GROUP.mkv", title: "Example Show", episode: 3, season: 2,
			metadata: polochon.VideoMetadata{Quality: polochon.Quality1080p, VideoCodec: "H.265", ReleaseGroup: "GROUP", Container: "mkv"},
		},
		{
			name: "Example.2024.MULTi.720p.x264-GROUP.mp4", title: "Example", year: 2024,
			metadata: polochon.VideoMetadata{Quality: polochon.Quality720p, VideoCodec: "H.264", ReleaseGroup: "GROUP", Container: "mp4"},
		},
		{
			name: "Example.2024.FRE.FRENCH.ENG.1080p.WEB-DL.x264-GROUP.mkv", title: "Example", year: 2024,
			metadata: polochon.VideoMetadata{Quality: polochon.Quality1080p, VideoCodec: "H.264", ReleaseGroup: "GROUP", Container: "mkv"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Metadata in the parent directory must not affect the guess.
			file := polochon.File{Path: filepath.Join("Other.Show.S09E99.480p", tc.name), Size: 123}
			g := &Guessit{}
			video, err := g.Guess(file, movieConf, showConf)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(*video.GetFile(), file) {
				t.Fatalf("file not preserved: %+v", video.GetFile())
			}
			var metadata polochon.VideoMetadata
			switch v := video.(type) {
			case *polochon.Movie:
				if tc.episode != 0 || v.Title != tc.title || v.Year != tc.year || !reflect.DeepEqual(v.MovieConfig, movieConf) {
					t.Fatalf("unexpected movie: %+v", v)
				}
				metadata = v.VideoMetadata
			case *polochon.ShowEpisode:
				if tc.episode == 0 || v.ShowTitle != tc.title || v.Episode != tc.episode || v.Season != tc.season || v.Show.Title != tc.title || v.Show.Year != tc.year || !reflect.DeepEqual(v.ShowConfig, showConf) {
					t.Fatalf("unexpected episode: %+v", v)
				}
				metadata = v.VideoMetadata
			default:
				t.Fatalf("unexpected video type: %T", video)
			}
			if !reflect.DeepEqual(metadata, tc.metadata) {
				t.Fatalf("metadata = %+v, want %+v", metadata, tc.metadata)
			}
			got, err := g.GuessMetadata(&file)
			if err != nil || !reflect.DeepEqual(*got, tc.metadata) {
				t.Fatalf("GuessMetadata = %+v, %v; want %+v", got, err, tc.metadata)
			}
		})
	}
}

func TestGuessRejectsCollectionsAndMissingIdentity(t *testing.T) {
	for _, name := range []string{
		"", "Show S01", "Show S01.mkv", "Show Season 1 Complete 1080p x265-GROUP",
		"Show.S01-S04.1080p.x265-GROUP", "Show.S01E01-08.1080p.x265-GROUP",
	} {
		t.Run(name, func(t *testing.T) {
			video, err := (&Guessit{}).Guess(polochon.File{Path: name}, polochon.MovieConfig{}, polochon.ShowConfig{})
			if err == nil || video != nil {
				t.Fatalf("Guess = %+v, %v; want an error and no video", video, err)
			}
		})
	}
}

func TestGuessMetadataFromSeason(t *testing.T) {
	got, err := (&Guessit{}).GuessMetadata(&polochon.File{Path: "Show Season 1 Complete FRE ENG 4K x265-GROUP"})
	if err != nil || got.Quality != polochon.Quality2160p || len(got.EmbeddedSubtitles) != 0 {
		t.Fatalf("unexpected season metadata: %+v, %v", got, err)
	}
}

func TestGuessVideoRangeRetainsFirstEpisode(t *testing.T) {
	// The upstream video model has no episode list; do not imply full coverage.
	video, err := (&Guessit{}).Guess(polochon.File{Path: "Show.S01E02-03.720p.x264-GROUP.mkv"}, polochon.MovieConfig{}, polochon.ShowConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := video.(*polochon.ShowEpisode); !ok || e.Season != 1 || e.Episode != 2 {
		t.Fatalf("unexpected range guess: %+v", video)
	}
}
