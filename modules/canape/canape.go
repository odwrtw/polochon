package canape

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"gopkg.in/yaml.v2"

	polochon "github.com/odwrtw/polochon/lib"
)

// Make sure that the module is a wishlister
var _ polochon.Wishlister = (*Wishlist)(nil)

// Register a new Subtitiler
func init() {
	polochon.RegisterModule(&Wishlist{})
}

// Module constants
const (
	moduleName = "canape"
)

// UserWishlist represents the configurations to get a user wishlist
type UserWishlist struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

type movieResponse struct {
	Status string                 `json:"status"`
	Movies []polochon.WishedMovie `json:"data"`
}

type showResponse struct {
	Status string                `json:"status"`
	Shows  []polochon.WishedShow `json:"data"`
}

// Wishlist holds the canape wishlists
type Wishlist struct {
	*Params
	configured bool
	log        *slog.Logger
}

// Params represents the module params
type Params struct {
	Wishlists []UserWishlist `yaml:"wishlists"`
}

// Init implements the module interface
func (w *Wishlist) Init(p []byte, log *slog.Logger) error {
	if w.configured {
		return nil
	}

	w.log = log.With("module", moduleName)

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return w.InitWithParams(params)
}

// InitWithParams configures the module
func (w *Wishlist) InitWithParams(params *Params) error {
	w.Params = params
	return nil
}

// Name implements the Module interface
func (w *Wishlist) Name() string {
	return moduleName
}

// Status implements the Module interface
func (w *Wishlist) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

// GetMovieWishlist gets the movies wishlist
func (w *Wishlist) GetMovieWishlist(ctx context.Context) ([]*polochon.WishedMovie, error) {
	wl, err := w.getMovieWishlists(ctx)
	if err != nil {
		return nil, err
	}

	return wl.Movies, nil
}

// GetShowWishlist gets the show wishlist
func (w *Wishlist) GetShowWishlist(ctx context.Context) ([]*polochon.WishedShow, error) {
	wl, err := w.getShowWishlists(ctx)
	if err != nil {
		return nil, err
	}

	return wl.Shows, nil
}

// Get all the users movie wishlists
func (w *Wishlist) getMovieWishlists(ctx context.Context) (*polochon.Wishlist, error) {
	wl := &polochon.Wishlist{}

	for _, userWishlist := range w.Wishlists {
		movies, err := userWishlist.getMovieWishlist(ctx)
		if err != nil {
			return nil, err
		}

		// Add the movies
		for _, movie := range movies {
			if err := wl.AddMovie(&movie); err != nil {
				return nil, err
			}
		}
	}

	return wl, nil
}

// Get all the users show wishlists
func (w *Wishlist) getShowWishlists(ctx context.Context) (*polochon.Wishlist, error) {
	wl := &polochon.Wishlist{}

	for _, userWishlist := range w.Wishlists {
		showList, err := userWishlist.getShowWishlist(ctx)
		if err != nil {
			return nil, err
		}

		// Add the shows
		for _, s := range showList {
			if err := wl.AddShow(&s); err != nil {
				return nil, err
			}
		}
	}

	return wl, nil
}

// Get a user's show wishlist
func (w *UserWishlist) getShowWishlist(ctx context.Context) ([]polochon.WishedShow, error) {
	wishlist := &showResponse{}
	err := w.request(ctx, "wishlist/shows", wishlist)
	if err != nil {
		return nil, err
	}
	return wishlist.Shows, nil
}

// Get a user's movie wishlist
func (w *UserWishlist) getMovieWishlist(ctx context.Context) ([]polochon.WishedMovie, error) {
	wishlist := &movieResponse{}
	err := w.request(ctx, "wishlist/movies", wishlist)
	if err != nil {
		return nil, err
	}
	return wishlist.Movies, nil

}

func (w *UserWishlist) request(ctx context.Context, URL string, response any) error {
	// Create a new request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s", w.URL, URL), nil)
	if err != nil {
		return err
	}

	// Add the token to the request
	params := url.Values{}
	params.Set("token", w.Token)
	req.URL.RawQuery = params.Encode()

	// Add the headers
	req.Header.Add("Content-type", "application/json")

	// Get the page
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("canape: invalid http code %q", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(response)
}
