package pam

import (
	"github.com/odwrtw/papi"
	"github.com/odwrtw/polochon/lib"
)

func episodePolochonToPapi(episode *polochon.ShowEpisode) (*papi.Episode, error) {
	if episode.ShowImdbID == "" {
		return nil, ErrMissingImdbID
	}

	if episode.Season == 0 || episode.Episode == 0 {
		return nil, ErrMissingEpisodeOrSeason
	}

	return &papi.Episode{
		ShowImdbID: episode.ShowImdbID,
		Season:     episode.Season,
		Episode:    episode.Episode,
	}, nil
}

func episodePapiToPolochon(papiEpisode *papi.Episode, polochonEpisode *polochon.ShowEpisode) {
	polochonEpisode.Season = papiEpisode.Season
	polochonEpisode.Episode = papiEpisode.Episode
	polochonEpisode.Title = papiEpisode.Title
	polochonEpisode.ShowTitle = papiEpisode.Title
	polochonEpisode.TvdbID = papiEpisode.TvdbID
	polochonEpisode.Aired = papiEpisode.Aired
	polochonEpisode.Plot = papiEpisode.Plot
	polochonEpisode.Runtime = papiEpisode.Runtime
	polochonEpisode.Thumb = papiEpisode.Thumb
	polochonEpisode.Rating = papiEpisode.Rating
	polochonEpisode.ShowImdbID = papiEpisode.ShowImdbID
	polochonEpisode.ShowTvdbID = papiEpisode.ShowTvdbID
	polochonEpisode.EpisodeImdbID = papiEpisode.EpisodeImdbID
	polochonEpisode.ReleaseGroup = papiEpisode.ReleaseGroup
}

func (p *Pam) getEpisodeDetails(episode *polochon.ShowEpisode) error {
	papiEpisode, err := episodePolochonToPapi(episode)
	if err != nil {
		return err
	}

	// Get the details of the resource
	if err := p.client.GetDetails(papiEpisode); err != nil {
		return err
	}

	episodePapiToPolochon(papiEpisode, episode)

	return nil
}
