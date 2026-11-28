package downloader

import (
	"context"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/odwrtw/whatsthis"

	polochon "github.com/odwrtw/polochon/lib"
)

// calendarSeason holds the deduplicated calendar episodes of a season, sorted
// by episode number
type calendarSeason struct {
	number   int
	episodes []*polochon.ShowCalendarEpisode
}

// calendarSeasons groups the calendar by season, sorted, skipping specials.
// Duplicate episodes keep the latest known air date.
func calendarSeasons(calendar *polochon.ShowCalendar) []calendarSeason {
	seasons := map[int]map[int]*polochon.ShowCalendarEpisode{}
	for _, e := range calendar.Episodes {
		if e == nil || e.Season == 0 {
			continue
		}

		episodes, ok := seasons[e.Season]
		if !ok {
			episodes = map[int]*polochon.ShowCalendarEpisode{}
			seasons[e.Season] = episodes
		}

		prev, ok := episodes[e.Episode]
		if ok && (e.AiredDate == nil || (prev.AiredDate != nil && !e.AiredDate.After(*prev.AiredDate))) {
			continue
		}
		episodes[e.Episode] = e
	}

	out := make([]calendarSeason, 0, len(seasons))
	for _, number := range slices.Sorted(maps.Keys(seasons)) {
		episodes := seasons[number]
		season := calendarSeason{number: number}
		for _, n := range slices.Sorted(maps.Keys(episodes)) {
			season.episodes = append(season.episodes, episodes[n])
		}
		out = append(out, season)
	}
	return out
}

// hasAired reports whether an episode has a known air date in the past
func hasAired(e *polochon.ShowCalendarEpisode) bool {
	return e.AiredDate != nil && !e.AiredDate.IsZero() && e.AiredDate.Before(time.Now())
}

// qualityRank returns the position of a quality in the wanted list
func qualityRank(q polochon.Quality, qualities []polochon.Quality) int {
	if i := slices.Index(qualities, q); i >= 0 {
		return i
	}
	return len(qualities)
}

// queuedKey identifies a show torrent of the download client, a season pack
// having no episode
type queuedKey struct {
	imdbID          string
	season, episode int
}

// queuedShowTorrents returns the season packs and episodes already in the
// download client so they are not queued again while downloading
func (d *Downloader) queuedShowTorrents(log *slog.Logger) map[queuedKey]bool {
	torrents, err := d.config.Downloader.Client.List()
	if err != nil {
		log.Error("failed to list the queued torrents", "error", err)
		return nil
	}

	queued := map[queuedKey]bool{}
	for _, t := range torrents {
		if t.Type == polochon.TypeSeason || t.Type == polochon.TypeEpisode {
			queued[queuedKey{t.ImdbID, t.Season, t.Episode}] = true
		}
	}
	return queued
}

func (d *Downloader) downloadMissingSeason(ctx context.Context, log *slog.Logger, s *polochon.Show, wishedShow *polochon.WishedShow, season calendarSeason, queued map[queuedKey]bool) {
	log = log.With("season", season.number)

	if queued[queuedKey{wishedShow.ImdbID, season.number, 0}] {
		log.Debug("season pack already queued")
		return
	}

	var tracked, missing []*polochon.ShowCalendarEpisode
	for _, e := range season.episodes {
		if e.IsOlder(wishedShow) {
			continue
		}
		tracked = append(tracked, e)

		if !e.IsAvailable() || queued[queuedKey{wishedShow.ImdbID, season.number, e.Episode}] {
			continue
		}

		ok, err := d.library.HasShowEpisode(wishedShow.ImdbID, season.number, e.Episode)
		if err != nil {
			log.Error(err.Error(), "episode", e.Episode)
			continue
		}
		if !ok {
			missing = append(missing, e)
		}
	}

	if len(missing) == 0 {
		return
	}

	var pack *polochon.Torrent
	if len(missing)*2 > len(season.episodes) {
		pack = findSeasonPack(ctx, log, s, season, tracked, wishedShow.Qualities)
	}

	// No single episode can beat a pack in the preferred quality
	if pack != nil && qualityRank(pack.Quality, wishedShow.Qualities) == 0 {
		d.downloadSeasonPack(log, wishedShow, season.number, pack)
		return
	}

	var episodes []*polochon.Torrent
	for _, e := range missing {
		if torrent := d.chooseEpisodeTorrent(ctx, log, s, wishedShow, season.number, e.Episode); torrent != nil {
			episodes = append(episodes, torrent)
		}
	}

	if pack != nil {
		packRank := qualityRank(pack.Quality, wishedShow.Qualities)
		if !slices.ContainsFunc(episodes, func(t *polochon.Torrent) bool {
			return qualityRank(t.Quality, wishedShow.Qualities) < packRank
		}) {
			d.downloadSeasonPack(log, wishedShow, season.number, pack)
			return
		}
		log.Debug("single episodes have a better quality than the season pack")
	}

	for _, torrent := range episodes {
		d.download(log, torrent)
	}
}

// downloadSeasonPack submits a pack labelled with the show IMDb ID and season
func (d *Downloader) downloadSeasonPack(log *slog.Logger, wishedShow *polochon.WishedShow, season int, pack *polochon.Torrent) {
	log.Info("downloading season pack", "name", pack.Result.Name, "quality", pack.Quality)
	d.download(log, &polochon.Torrent{
		ImdbID:  wishedShow.ImdbID,
		Type:    polochon.TypeSeason,
		Season:  season,
		Quality: pack.Quality,
		Result:  pack.Result,
	})
}

// findSeasonPack asks each show torrenter in order for the season and returns
// the best pack of the first torrenter having one
func findSeasonPack(ctx context.Context, log *slog.Logger, s *polochon.Show, season calendarSeason, tracked []*polochon.ShowCalendarEpisode, qualities []polochon.Quality) *polochon.Torrent {
	if s.Title == "" {
		return nil
	}

	for _, torrenter := range s.GetTorrenters() {
		ss := &polochon.ShowSeason{ShowImdbID: s.ImdbID, ShowTitle: s.Title, Season: season.number}
		if err := torrenter.GetTorrents(ctx, ss); err != nil {
			log.Debug("no season pack", "torrenter", torrenter.Name(), "error", err)
			continue
		}

		var candidates []*polochon.Torrent
		for _, t := range ss.Torrents {
			if isSeasonPack(t, s, season, tracked) {
				candidates = append(candidates, t)
			}
		}

		if pack := polochon.ChooseTorrentFromQualities(polochon.FilterTorrents(candidates), qualities); pack != nil {
			return pack
		}
	}

	return nil
}

// isSeasonPack reports whether a search result is a pack of exactly the
// tracked episodes of the season, all aired. It sets a missing quality from
// the parsed name.
func isSeasonPack(t *polochon.Torrent, s *polochon.Show, season calendarSeason, tracked []*polochon.ShowCalendarEpisode) bool {
	if t == nil || t.Result == nil || t.Result.URL == "" || t.Result.Seeders <= 0 {
		return false
	}

	// Episode ranges may be typed as an episode; Episodes is only set for
	// ranges, so single episodes are still rejected.
	info := whatsthis.File(t.Result.Name)
	if info.Type != whatsthis.ShowSeason && (info.Type != whatsthis.Episode || len(info.Episodes) == 0) {
		return false
	}
	if info.Season != season.number {
		return false
	}
	if !strings.EqualFold(info.Title, s.Title) {
		return false
	}
	if info.Year != 0 && s.Year != 0 && info.Year != s.Year {
		return false
	}

	if len(info.Episodes) == 0 {
		// A plain season pack is presumed to hold every known episode
		if len(tracked) != len(season.episodes) {
			return false
		}
	} else if !slices.EqualFunc(info.Episodes, tracked, func(n int, e *polochon.ShowCalendarEpisode) bool { return n == e.Episode }) {
		return false
	}

	if slices.ContainsFunc(tracked, func(e *polochon.ShowCalendarEpisode) bool { return !hasAired(e) }) {
		return false
	}

	if t.Quality == "" {
		if q, err := polochon.StringToQuality(info.ScreenSize); err == nil {
			t.Quality = *q
		}
	}
	return true
}
