package polochon

import (
	"log"

	"github.com/Sirupsen/logrus"
)

// Wishlist represents a user wishlist
type Wishlist struct {
	WishlistConfig `xml:"-" json:"-"`
	log            *logrus.Entry
	Movies         []*WishedMovie `json:"movies"`
	Shows          []*WishedShow  `json:"shows"`
}

// NewWishlist returns a new wishlist
func NewWishlist(wishlistConfig WishlistConfig, log *logrus.Logger) *Wishlist {
	logEntry := log.WithField("type", "wishlist")
	return &Wishlist{
		WishlistConfig: wishlistConfig,
		log:            logEntry,
	}
}

// Fetch the infomations from the wishlister an returns a merged wishlist
// TODO: merge the wishlists from the different wishlisters
func (w *Wishlist) Fetch() error {
	// Movie wishlists
	for _, wl := range w.Wishlisters {
		m, err := wl.GetMovieWishlist()
		if err != nil {
			w.log.Warnf("failed to get movie wishlist from wishlister: %q", err)
			continue
		}

		w.setDefaultMovieQualities(m)
		w.Movies = append(w.Movies, m...)
	}

	// Shows wishlists
	for _, wl := range w.Wishlisters {
		m, err := wl.GetShowWishlist()
		if err != nil {
			w.log.Warnf("failed to get show wishlist from wishlister: %q", err)
			continue
		}

		w.setDefaultShowQualities(m)
		w.Shows = append(w.Shows, m...)
	}

	return nil
}

func (w *Wishlist) setDefaultMovieQualities(movies []*WishedMovie) {
	// No need to go further if there is no config
	if len(w.MovieDefaultQualities) == 0 {
		return
	}

	for _, m := range movies {
		if len(m.Qualities) > 0 {
			continue
		}
		m.Qualities = w.MovieDefaultQualities
	}
}

func (w *Wishlist) setDefaultShowQualities(shows []*WishedShow) {
	// No need to go further if there is no config
	if len(w.ShowDefaultQualities) == 0 {
		return
	}

	for _, s := range shows {
		if len(s.Qualities) > 0 {
			continue
		}
		s.Qualities = w.ShowDefaultQualities
	}
}

// Wishlister is an interface which defines the behavior of the wishlister
// modules
type Wishlister interface {
	GetMovieWishlist() ([]*WishedMovie, error)
	GetShowWishlist() ([]*WishedShow, error)
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

// RegisterWishlister helps register a new Wishlister
func RegisterWishlister(name string, f func(params map[string]interface{}, log *logrus.Entry) (Wishlister, error)) {
	if _, ok := registeredModules.Wishlisters[name]; ok {
		log.Panicf("modules: %q of type %q is already registered", name, TypeWishlister)
	}

	// Register the module
	registeredModules.Wishlisters[name] = f
}
