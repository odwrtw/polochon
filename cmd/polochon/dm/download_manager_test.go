package dm

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/odwrtw/polochon/modules/guessit"
)

const packShowID = "tt1234567"

type packDetailer struct {
	polochon.Module
	t      *testing.T
	images string
	fail   bool
	calls  int
}

func (d *packDetailer) Name() string { return "pack_test" }

func (d *packDetailer) GetDetails(_ context.Context, value any) error {
	d.calls++
	v, ok := value.(*polochon.ShowEpisode)
	if !ok || v.ShowImdbID != packShowID || v.Show == nil || v.Show.ImdbID != packShowID {
		d.t.Fatalf("detailer did not receive the labelled show: %+v", value)
	}
	if d.fail {
		return polochon.ErrGettingDetails
	}
	v.ShowTitle = "Show"
	v.Title = "Episode"
	v.Show = &polochon.Show{
		ImdbID: packShowID, Title: "Show", Year: 2026,
		BannerURL: d.images, FanartURL: d.images, PosterURL: d.images,
	}
	return nil
}

type packDownloader struct {
	polochon.Module
	torrent *polochon.Torrent
	removed int
}

func (d *packDownloader) Download(*polochon.Torrent) error { return nil }
func (d *packDownloader) List() ([]*polochon.Torrent, error) {
	return []*polochon.Torrent{d.torrent}, nil
}
func (d *packDownloader) Remove(*polochon.Torrent) error {
	d.removed++
	return nil
}

func newPackManager(t *testing.T, paths []string) (*DownloadManager, *packDetailer, *packDownloader) {
	t.Helper()
	images := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("image"))
	}))
	t.Cleanup(images.Close)

	root := t.TempDir()
	detailer := &packDetailer{t: t, images: images.URL}
	client := &packDownloader{torrent: &polochon.Torrent{
		ImdbID: packShowID,
		Type:   polochon.TypeSeason,
		Season: 1,
		Status: &polochon.TorrentStatus{ID: "1", IsFinished: true, FilePaths: paths},
	}}
	config := &configuration.Config{
		Watcher:         configuration.WatcherConfig{Dir: filepath.Join(root, "watcher")},
		DownloadManager: configuration.DownloadManagerConfig{Dir: filepath.Join(root, "downloads")},
		Downloader:      configuration.DownloaderConfig{Enabled: true, Client: client},
		Library:         configuration.LibraryConfig{MovieDir: filepath.Join(root, "movies"), ShowDir: filepath.Join(root, "shows")},
		Show:            polochon.ShowConfig{Detailers: []polochon.Detailer{detailer}},
		File: polochon.FileConfig{
			VideoExtensions:       []string{".mkv"},
			ExcludeFileContaining: []string{"sample"},
			Guessers:              []polochon.Guesser{&guessit.Guessit{}},
		},
	}
	for _, dir := range []string{config.Watcher.Dir, config.DownloadManager.Dir, config.Library.MovieDir, config.Library.ShowDir} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range paths {
		path := filepath.Join(config.DownloadManager.Dir, p)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("video"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(config, library.New(config), log), detailer, client
}

func TestSeasonPack(t *testing.T) {
	paths := []string{"pack/Show.S01E01.mkv", "pack/nested/Show.S01E02.mkv", "pack/Show.S01E01.sample.mkv", "pack/readme.txt"}
	manager, detailer, client := newPackManager(t, paths)

	manager.run(context.Background())
	for _, e := range []int{1, 2} {
		if ok, err := manager.library.HasShowEpisode(packShowID, 1, e); err != nil || !ok {
			t.Fatalf("episode %d not in library: %v, %v", e, ok, err)
		}
	}
	if !polochon.NewFile(filepath.Join(manager.config.DownloadManager.Dir, paths[0])).IsSymlink() {
		t.Fatal("organized episode was not left as a symlink for seeding")
	}
	if client.removed != 0 {
		t.Fatal("torrent removed on the run organizing it")
	}

	// The next run only cleans the seeded torrent
	manager.run(context.Background())
	if detailer.calls != 2 {
		t.Fatalf("detailer called %d times, want 2", detailer.calls)
	}
	if client.removed != 1 {
		t.Fatalf("removed %d torrents, want 1", client.removed)
	}
}

func TestSeasonPackPartialFailureDropsSymlinks(t *testing.T) {
	paths := []string{"pack/Show.S01E01.mkv", "pack/Show.S02E01.mkv"}
	manager, _, _ := newPackManager(t, paths)

	manager.run(context.Background())
	if _, err := os.Lstat(filepath.Join(manager.config.Watcher.Dir, paths[0])); !os.IsNotExist(err) {
		t.Fatalf("organized episode symlink moved to the watcher: %v", err)
	}
	if !polochon.NewFile(filepath.Join(manager.config.Watcher.Dir, paths[1])).Exists() {
		t.Fatal("unorganized episode not moved to the watcher directory")
	}
}

func TestSeasonPackFallsBackToWatcher(t *testing.T) {
	for _, tc := range []struct {
		name string
		path string
		fail bool
	}{
		{name: "other season", path: "pack/Show.S02E01.mkv"},
		{name: "not an episode", path: "pack/Movie.2001.mkv"},
		{name: "details failure", path: "pack/Show.S01E01.mkv", fail: true},
		{name: "no video file", path: "pack/Show.S01.rar"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager, detailer, client := newPackManager(t, []string{tc.path})
			detailer.fail = tc.fail

			manager.run(context.Background())
			if client.removed != 1 {
				t.Fatalf("removed %d torrents, want 1", client.removed)
			}
			if !polochon.NewFile(filepath.Join(manager.config.Watcher.Dir, tc.path)).Exists() {
				t.Fatal("pack not moved to the watcher directory")
			}
		})
	}
}
