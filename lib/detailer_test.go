package polochon

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type detailingStub struct {
	name string
	get  func(context.Context, any) error
}

func (d *detailingStub) Name() string                                { return d.name }
func (d *detailingStub) Init([]byte, *slog.Logger) error             { return nil }
func (d *detailingStub) Status() (ModuleStatus, error)               { return StatusOK, nil }
func (d *detailingStub) GetDetails(ctx context.Context, v any) error { return d.get(ctx, v) }

type continuingStub struct {
	*detailingStub
	keepGoing bool
}

func (d *continuingStub) ContinueOnSuccess() bool { return d.keepGoing }

func detailingLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func partialDetailer(get func(context.Context, any) error) Detailer {
	return &continuingStub{detailingStub: &detailingStub{name: "partial", get: get}, keepGoing: true}
}

func TestGetDetailsOrdinaryBehavior(t *testing.T) {
	for _, withCapability := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "continuation disabled"}[withCapability], func(t *testing.T) {
			calls := 0
			first := &detailingStub{name: "first", get: func(_ context.Context, v any) error {
				calls++
				v.(*Movie).Title = "changed on failure"
				return ErrGettingDetails
			}}
			second := &detailingStub{name: "second", get: func(_ context.Context, v any) error {
				calls++
				if v.(*Movie).Title != "changed on failure" {
					t.Fatal("ordinary failure mutations should retain existing semantics")
				}
				v.(*Movie).Plot = "complete"
				return nil
			}}
			var successful Detailer = second
			if withCapability {
				successful = &continuingStub{detailingStub: second}
			}
			last := &detailingStub{name: "last", get: func(context.Context, any) error {
				t.Fatal("must stop on ordinary success")
				return nil
			}}
			m := &Movie{MovieConfig: MovieConfig{Detailers: []Detailer{first, successful, last}}}
			if err := GetDetails(context.Background(), m, detailingLog()); err != nil {
				t.Fatal(err)
			}
			if calls != 2 || m.Plot != "complete" {
				t.Fatalf("calls=%d movie=%+v", calls, m)
			}
		})
	}
}

func TestGetDetailsContinuesAndPreservesVideo(t *testing.T) {
	m := &Movie{Title: "alias", Genres: []string{"Action"}}
	m.Path = "movie.mkv"
	m.Quality = Quality720p
	m.EmbeddedSubtitles = []Language{"en_US"}
	m.Subtitles = []*Subtitle{{Video: m, Data: []byte("subtitle")}}
	m.Torrents = []*Torrent{{Result: &TorrentResult{Name: "torrent"}, Status: &TorrentStatus{FilePaths: []string{"movie.mkv"}}}}
	subtitle, torrent := m.Subtitles[0], m.Torrents[0]
	m.Detailers = []Detailer{
		partialDetailer(func(_ context.Context, v any) error {
			trial := v.(*Movie)
			trial.ImdbID = "tt1"
			trial.Title = "canonical"
			return nil
		}),
		&detailingStub{name: "provider", get: func(_ context.Context, v any) error {
			trial := v.(*Movie)
			if trial.ImdbID != "tt1" || trial.Title != "canonical" {
				t.Fatalf("partial details missing: %+v", trial)
			}
			trial.Plot = "complete"
			return nil
		}},
	}
	if err := GetDetails(context.Background(), m, detailingLog()); err != nil {
		t.Fatal(err)
	}
	if m.Subtitles[0] != subtitle || m.Torrents[0] != torrent ||
		m.Plot != "complete" || m.Path != "movie.mkv" || m.Quality != Quality720p ||
		string(m.Subtitles[0].Data) != "subtitle" || m.Subtitles[0].Video != m ||
		m.Torrents[0].Result.Name != "torrent" || m.Torrents[0].Status.FilePaths[0] != "movie.mkv" ||
		len(m.Detailers) != 2 || m.EmbeddedSubtitles[0] != "en_US" {
		t.Fatalf("video state lost: %+v", m)
	}
}

func TestGetDetailsPreservesFailureMutations(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "partial success", true: "partial failure"}[fail], func(t *testing.T) {
			m := &Movie{}
			m.Detailers = []Detailer{
				partialDetailer(func(_ context.Context, v any) error {
					v.(*Movie).ImdbID = "tt1"
					if fail {
						return ErrGettingDetails
					}
					return nil
				}),
				&detailingStub{name: "failed", get: func(_ context.Context, v any) error {
					v.(*Movie).TmdbID = 99
					return ErrGettingDetails
				}},
				&detailingStub{name: "success", get: func(_ context.Context, v any) error {
					if v != m || m.ImdbID != "tt1" || m.TmdbID != 99 {
						t.Fatalf("in-place mutations lost: %+v", m)
					}
					return nil
				}},
			}
			if err := GetDetails(context.Background(), m, detailingLog()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetDetailsDoesNotRetry(t *testing.T) {
	calls := 0
	m := &Movie{MovieConfig: MovieConfig{Detailers: []Detailer{
		partialDetailer(func(_ context.Context, v any) error {
			v.(*Movie).ImdbID = "tt1"
			return nil
		}),
		&detailingStub{name: "failed", get: func(context.Context, any) error {
			calls++
			return ErrGettingDetails
		}},
	}}}
	if err := GetDetails(context.Background(), m, detailingLog()); !errors.Is(err, ErrGettingDetails) {
		t.Fatalf("got %v, want ErrGettingDetails", err)
	}
	if calls != 1 || m.ImdbID != "tt1" {
		t.Fatalf("calls=%d movie=%+v", calls, m)
	}
}

func TestGetDetailsPartialIsNotCompletion(t *testing.T) {
	show := &Show{ShowConfig: ShowConfig{Detailers: []Detailer{
		partialDetailer(func(_ context.Context, v any) error {
			v.(*Show).Title = "partial"
			return nil
		}),
	}}}
	if err := GetDetails(context.Background(), show, detailingLog()); !errors.Is(err, ErrGettingDetails) {
		t.Fatalf("partial success completed enrichment: %v", err)
	}
	if show.Title != "partial" {
		t.Fatal("partial details should remain in place")
	}
}

func TestGetDetailsEpisodePreservesParent(t *testing.T) {
	show := &Show{Title: "alias"}
	ep := &ShowEpisode{Show: show, ShowTitle: "alias", Season: 1, Episode: 2}
	show.Episodes = []*ShowEpisode{ep}
	ep.Detailers = []Detailer{
		partialDetailer(func(_ context.Context, v any) error {
			trial := v.(*ShowEpisode)
			trial.ShowImdbID = "tt1"
			trial.Show.ImdbID = "tt1"
			return nil
		}),
		&detailingStub{name: "provider", get: func(_ context.Context, v any) error {
			trial := v.(*ShowEpisode)
			if trial.Show.Episodes[0] != trial {
				t.Fatal("parent cycle lost")
			}
			trial.Show.Plot = "show plot"
			trial.Plot = "episode plot"
			return nil
		}},
	}
	if err := GetDetails(context.Background(), ep, detailingLog()); err != nil {
		t.Fatal(err)
	}
	if ep.Show != show || show.Episodes[0] != ep || show.Plot != "show plot" || ep.Plot != "episode plot" || ep.Season != 1 || ep.Episode != 2 {
		t.Fatalf("lost parent/cycle: episode=%+v show=%+v", ep, show)
	}
}

func TestGetDetailsCancellation(t *testing.T) {
	for _, continuing := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary", true: "partial"}[continuing], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			m := &Movie{}
			if continuing {
				m.Detailers = append(m.Detailers, partialDetailer(func(_ context.Context, v any) error {
					v.(*Movie).ImdbID = "tt1"
					return nil
				}))
			}
			m.Detailers = append(m.Detailers, &detailingStub{name: "cancel", get: func(context.Context, any) error {
				calls++
				cancel()
				return nil
			}})
			if err := GetDetails(ctx, m, detailingLog()); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation lost: %v", err)
			}
			if calls != 1 {
				t.Fatalf("unexpected retry: %d", calls)
			}
		})
	}
}

func TestGetDetailsCatalogDeadlineDoesNotCancelProviders(t *testing.T) {
	calls := 0
	m := &Movie{MovieConfig: MovieConfig{Detailers: []Detailer{
		partialDetailer(func(context.Context, any) error { return context.DeadlineExceeded }),
		&detailingStub{name: "provider", get: func(context.Context, any) error {
			calls++
			return nil
		}},
	}}}
	if err := GetDetails(context.Background(), m, detailingLog()); err != nil || calls != 1 {
		t.Fatalf("child deadline blocked provider fallback: calls=%d error=%v", calls, err)
	}
}

func TestGetDetailsAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m := &Movie{MovieConfig: MovieConfig{Detailers: []Detailer{
		partialDetailer(func(context.Context, any) error {
			t.Fatal("canceled operation called a detailer")
			return nil
		}),
	}}}
	if err := GetDetails(ctx, m, detailingLog()); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}

func TestGetDetailsNoDetailers(t *testing.T) {
	if err := GetDetails(context.Background(), &Movie{}, detailingLog()); !errors.Is(err, ErrGettingDetails) {
		t.Fatal(err)
	}
}
