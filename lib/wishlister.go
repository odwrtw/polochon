package polochon

import (
	"fmt"

	"github.com/Sirupsen/logrus"
)

// Wishlister is an interface which defines the behavior of the wishlister
// modules
type Wishlister interface {
	GetMovieWishlist(*logrus.Entry) ([]*WishedMovie, error)
	GetShowWishlist(*logrus.Entry) ([]*WishedShow, error)
}

// WishedMovie represents a wished movie and its expected qualities
type WishedMovie struct {
	ImdbID    string    `json:"imdb_id"`
	Qualities []Quality `json:"qualities"`
}

// WishedShow represents a wished show, its expected qualities and the season /
// episode to start tracking
type WishedShow struct {
	ImdbID    string    `json:"imdb_id"`
	Season    int       `json:"episode_from"`
	Episode   int       `json:"season_from"`
	Qualities []Quality `json:"qualities"`
}

// WishlistConfig represents the wishlist configurations
type WishlistConfig struct {
	Wishlisters           []Wishlister
	ShowDefaultQualities  []Quality `yaml:"show_default_qualities"`
	MovieDefaultQualities []Quality `yaml:"movie_default_qualities"`
}

// Wishlist represents a user wishlist
type Wishlist struct {
	WishlistConfig `xml:"-" json:"-"`
	log            *logrus.Entry
	Movies         []*WishedMovie `json:"movies"`
	Shows          []*WishedShow  `json:"shows"`
}

// NewWishlist returns a new wishlist
func NewWishlist(wishlistConfig WishlistConfig, log *logrus.Entry) *Wishlist {
	return &Wishlist{
		WishlistConfig: wishlistConfig,
		log:            log.WithField("type", "wishlist"),
	}
}

// RegisterWishlister helps register a new Wishlister
func RegisterWishlister(name string, f func(params []byte) (Wishlister, error)) {
	if _, ok := registeredModules.Wishlisters[name]; ok {
		panic(fmt.Sprintf("modules: %q of type %q is already registered", name, TypeDetailer))
	}

	// Register the module
	registeredModules.Wishlisters[name] = f
}

// Fetch the infomations from the wishlister an returns a merged wishlist
// TODO: merge the wishlists from the different wishlisters
func (w *Wishlist) Fetch() error {
	// Movie wishlists
	if err := w.fetchMovies(); err != nil {
		return err
	}

	// Shows wishlists
	if err := w.fetchShows(); err != nil {
		return err
	}

	return nil
}

func (w *Wishlist) fetchMovies() error {
	for _, wl := range w.Wishlisters {
		movieWishlist, err := wl.GetMovieWishlist(w.log)
		if err != nil {
			w.log.Warnf("failed to get movie wishlist from wishlister: %q", err)
			continue
		}

		for _, m := range movieWishlist {
			if err := w.AddMovie(m); err != nil {
				return err
			}
		}
	}

	w.setDefaultMovieQualities()

	return nil
}

func (w *Wishlist) fetchShows() error {
	for _, wl := range w.Wishlisters {
		showWishlist, err := wl.GetShowWishlist(w.log)
		if err != nil {
			w.log.Warnf("failed to get show wishlist from wishlister: %q", err)
			continue
		}

		for _, s := range showWishlist {
			if err := w.AddShow(s); err != nil {
				return err
			}
		}

	}

	w.setDefaultShowQualities()

	return nil
}

func (w *Wishlist) setDefaultMovieQualities() {
	// No need to go further if there is no config
	if w.Movies == nil {
		return
	}

	for _, m := range w.Movies {
		if len(m.Qualities) > 0 {
			continue
		}
		m.Qualities = w.MovieDefaultQualities
	}
}

func (w *Wishlist) setDefaultShowQualities() {
	// No need to go further if there is no config
	if w.Shows == nil {
		return
	}

	for _, s := range w.Shows {
		if len(s.Qualities) > 0 {
			continue
		}
		s.Qualities = w.ShowDefaultQualities
	}
}

// AddMovie helps add a movie to the movie list, if the movie is already in the
// list, nothing happens
func (w *Wishlist) AddMovie(movie *WishedMovie) error {
	// Create an empty slice if there is no movies
	if w.Movies == nil {
		w.Movies = []*WishedMovie{}
	}

	// Check if the movie as already been added
	for _, m := range w.Movies {
		if movie.ImdbID == m.ImdbID {
			return nil
		}
	}

	w.Movies = append(w.Movies, &WishedMovie{
		ImdbID: movie.ImdbID,
	})

	return nil
}

// AddShow adds a show to the show list, if the show is already in the list,
// the oldest is kept or added
func (w *Wishlist) AddShow(show *WishedShow) error {
	// Create an empty slice if there is no shows
	if w.Shows == nil {
		w.Shows = []*WishedShow{}
	}

	// Check if the show as already been added
	for _, s := range w.Shows {
		if show.ImdbID != s.ImdbID {
			continue
		}

		// Do not treat empty data as valid data
		if show.Episode == 0 && show.Season == 0 {
			return nil
		}

		// If the show added and the current show have the same season number
		if show.Season == s.Season {
			// Older show is better
			if show.Episode < s.Episode {
				s.Episode = show.Episode
			}
		}

		// Older season is better
		if show.Season < s.Season {
			s.Season = show.Season
			s.Episode = show.Episode
		}

		// Done with this show
		return nil
	}

	// Nothing found let's add it
	w.Shows = append(w.Shows, &WishedShow{
		ImdbID:  show.ImdbID,
		Season:  show.Season,
		Episode: show.Episode,
	})

	return nil
}
