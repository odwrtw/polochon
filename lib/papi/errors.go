package papi

import "errors"

// Custom errors
var (
	ErrResourceNotFound               = errors.New("papi: resource not found")
	ErrNotImplemented                 = errors.New("papi: not implemented")
	ErrMissingMovie                   = errors.New("papi: missing movie")
	ErrMissingMovieID                 = errors.New("papi: missing movie id")
	ErrMissingShow                    = errors.New("papi: missing show")
	ErrMissingShowID                  = errors.New("papi: missing show id")
	ErrMissingSeason                  = errors.New("papi: missing season number")
	ErrMissingShowEpisodeInformations = errors.New("papi: missing show episode informations")
	ErrMissingShowImdbID              = errors.New("papi: missing show imdb id")
	ErrMissingTorrentData             = errors.New("papi: missing torrent data")
	ErrMissingTorrentURL              = errors.New("papi: missing torrent url")
	ErrMissingSubtitle                = errors.New("papi: missing subtitle")
	ErrMissingSubtitleVideo           = errors.New("papi: missing subtitle video")
)
