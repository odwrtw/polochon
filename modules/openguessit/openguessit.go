package openguessit

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/arbovm/levenshtein"
	"github.com/odwrtw/guessit"
	"github.com/odwrtw/polochon/lib"
	"github.com/oz/osdb"
)

// Video types
const (
	MovieType   = "movie"
	ShowType    = "episode"
	UnknownType = "unknown"
)

// Module constants
const (
	moduleName = "openguessit"
)

// Errors
var (
	ErrShowNameUnknown = errors.New("show title unknown")
)

// Register openguessit as a Guesser
func init() {
	polochon.RegisterGuesser(moduleName, NewFromRawYaml)
}

// OpenGuessit is a mix of opensubtitle and guessit
type OpenGuessit struct {
}

// New returns an new openguessit
func New() (polochon.Guesser, error) {
	return &OpenGuessit{}, nil
}

// NewFromRawYaml returns an new openguessit
func NewFromRawYaml(p []byte) (polochon.Guesser, error) {
	return New()
}

// Name implements the Module interface
func (og *OpenGuessit) Name() string {
	return moduleName
}

// Guess implements the Guesser interface
func (og *OpenGuessit) Guess(file polochon.File, movieConf polochon.MovieConfig, showConf polochon.ShowConfig, log *logrus.Entry) (polochon.Video, error) {
	g, err := NewGuesser(file.Path, log)
	if err != nil {
		return nil, fmt.Errorf("failed to get the file: %q", err)
	}

	guess, err := g.Guess()
	if err != nil {
		return nil, fmt.Errorf("failed to find the best guess: %q", err)
	}

	if guess == nil {
		return nil, fmt.Errorf("failed guess the file, nothing was found")
	}

	// Return the right video format
	switch guess.Type() {
	case ShowType:
		video := polochon.NewShowEpisodeFromFile(showConf, file)

		showTitle, err := guess.ShowName()
		if err != nil {
			return nil, err
		}
		video.ShowTitle = showTitle

		episode, err := strconv.Atoi(guess.Episode())
		if err != nil {
			return nil, err
		}
		video.Episode = episode

		season, err := strconv.Atoi(guess.Season())
		if err != nil {
			return nil, err
		}
		video.Season = season

		year, err := strconv.Atoi(guess.Year())
		if err != nil {
			return nil, err
		}

		// Add some infos on the show
		video.Show = &polochon.Show{
			ImdbID:    guess.ImdbID(),
			ShowTitle: showTitle,
			Title:     showTitle,
			Year:      year,
		}

		video.ReleaseGroup = guess.Guessit.ReleaseGroup

		return video, nil
	case MovieType:
		video := polochon.NewMovieFromFile(movieConf, file)

		video.Title = guess.MovieName()

		year, err := strconv.Atoi(guess.Year())
		if err != nil {
			return nil, err
		}
		video.Year = year

		return video, nil
	}

	return nil, errors.New("Failed to guess video type")
}

// Guesser represents a new guesser
type Guesser struct {
	FilePath     string
	Guessit      *Guessit
	OpenSubtitle []*OpenSubtitle
	log          *logrus.Entry
}

// NewGuesser returns an new guess
func NewGuesser(filePath string, log *logrus.Entry) (*Guesser, error) {
	g := &Guesser{
		FilePath:     filePath,
		OpenSubtitle: []*OpenSubtitle{},
		log:          log,
	}

	// Sometimes opensub returns an error... It should not be a problem,
	// guessit should be able to handle it alone
	if err := g.UpdateFromOpenSubtitle(); err != nil {
		g.log.Warningf("failed to get guess from opensubtitle: %q", err)
	}

	if err := g.UpdateFromGuessit(); err != nil {
		return nil, err
	}

	return g, nil
}

// Guess represents a (super) guess with a ranking
type Guess struct {
	Guessit      *Guessit
	OpenSubtitle *OpenSubtitle
	Ranking      float64
}

// Guessit represents the informations from guessit
type Guessit struct {
	Type         string
	MovieName    string
	ShowName     string
	Season       string
	Episode      string
	Year         string
	ReleaseGroup string
}

// OpenSubtitle represents the informations from OpenSubtitle
type OpenSubtitle struct {
	ImdbID    string
	Type      string
	MovieName string
	ShowName  string
	Season    string
	Episode   string
	Year      string
	LevDist   int
}

// Type returns the type of a guess
func (g *Guess) Type() string {
	// Returns opensub type if found
	if g.OpenSubtitle != nil {
		if g.OpenSubtitle.Type != "" {
			return g.OpenSubtitle.Type
		}
	}

	// Returns guessi type if found
	if g.Guessit != nil {
		if g.Guessit.Type != "" {
			return g.Guessit.Type
		}
	}

	return UnknownType
}

// ShowName returns the show name of a guess
func (g *Guess) ShowName() (string, error) {
	if g.OpenSubtitle != nil {
		if g.OpenSubtitle.ShowName != "" {
			return toUpperCaseFirst(g.OpenSubtitle.ShowName), nil
		}
	}

	if g.Guessit != nil {
		if g.Guessit.ShowName != "" {
			return toUpperCaseFirst(g.Guessit.ShowName), nil
		}
	}

	return "", ErrShowNameUnknown
}

// ImdbID returns the show imdbID of a guess
func (g *Guess) ImdbID() string {
	if g.OpenSubtitle != nil && g.OpenSubtitle.ImdbID != "" {
		return g.OpenSubtitle.ImdbID
	}

	return ""
}

// Year returns the show year of a guess
func (g *Guess) Year() string {
	if g.OpenSubtitle != nil {
		if g.OpenSubtitle.Year != "" {
			return g.OpenSubtitle.Year
		}
	}

	if g.Guessit != nil {
		if g.Guessit.Year != "" {
			return g.Guessit.Year
		}
	}

	return ""
}

// MovieName returns the movie name of a guess
func (g *Guess) MovieName() string {
	if g.OpenSubtitle != nil {
		if g.OpenSubtitle.MovieName != "" {
			return toUpperCaseFirst(g.OpenSubtitle.MovieName)
		}
	}

	if g.Guessit != nil {
		if g.Guessit.MovieName != "" {
			return toUpperCaseFirst(g.Guessit.MovieName)
		}
	}

	return ""
}

// Episode returns the episode number of a guess
func (g *Guess) Episode() string {
	if g.OpenSubtitle != nil {
		if g.OpenSubtitle.Episode != "" {
			return g.OpenSubtitle.Episode
		}
	}

	if g.Guessit != nil {
		if g.Guessit.Episode != "" {
			return g.Guessit.Episode
		}
	}

	return ""
}

// Season returns the season number of a guess
func (g *Guess) Season() string {
	if g.OpenSubtitle != nil {
		if g.OpenSubtitle.Season != "" {
			return g.OpenSubtitle.Season
		}
	}

	if g.Guessit != nil {
		if g.Guessit.Season != "" {
			return g.Guessit.Season
		}
	}

	return ""
}

// UpdateFromOpenSubtitle updates the guess with the OpenSubtitle informations
func (g *Guesser) UpdateFromOpenSubtitle() error {
	// Base path of the filename
	basePath := filepath.Base(g.FilePath)

	// OpenSubtitle client
	client, err := osdb.NewClient()
	if err != nil {
		return err
	}

	// Log in
	if err := client.LogIn("", "", "eng"); err != nil {
		return err
	}

	// Set the languages
	languages := []string{"eng"}

	// Hash movie file, and search
	subtitles, err := client.FileSearch(g.FilePath, languages)
	if err != nil {
		return err
	}

	// If nothing found, search by filename
	if len(subtitles) == 0 {
		client, err := osdb.NewClient()
		if err != nil {
			return err
		}

		err = client.LogIn("", "", "eng")
		if err != nil {
			return err
		}

		params := []interface{}{
			client.Token,
			[]map[string]string{
				{
					"query":         basePath,
					"sublanguageid": "en",
				},
			},
		}

		subtitles, err = client.SearchSubtitles(&params)
		if err != nil {
			return err
		}
	}

	// No subtitles found
	if len(subtitles) == 0 {
		return nil
	}

	for _, sub := range subtitles {
		switch sub.MovieKind {
		case MovieType:
			g.OpenSubtitle = append(g.OpenSubtitle, &OpenSubtitle{
				ImdbID:    fmt.Sprintf("tt%07s", sub.IDMovieImdb),
				Type:      MovieType,
				MovieName: sub.MovieFileName,
				Year:      sub.MovieYear,
				LevDist:   levenshtein.Distance(sub.SubFileName, basePath),
			})
		case ShowType:
			// The MovieFileName field returned by openSubtitles contains the
			// show name and the episode name.
			// e.g. `"Show Title" Show episode title`
			// Only the title is relevant
			showName := sub.MovieFileName
			s := strings.Split(sub.MovieFileName, `"`)
			if len(s) >= 2 {
				showName = s[1]
			}

			g.OpenSubtitle = append(g.OpenSubtitle, &OpenSubtitle{
				ImdbID:   fmt.Sprintf("tt%07s", sub.SeriesIMDBParent),
				Type:     ShowType,
				ShowName: showName,
				Season:   sub.SeriesSeason,
				Episode:  sub.SeriesEpisode,
				Year:     sub.MovieYear,
				LevDist:  levenshtein.Distance(sub.SubFileName, basePath),
			})
		default:
			return fmt.Errorf("Invalid movie kind: %q", sub.MovieKind)
		}
	}

	return nil
}

// UpdateFromGuessit updates the guess with the guessit informations
func (g *Guesser) UpdateFromGuessit() error {
	// Guesss the file infos from the name
	guess, err := guessit.Guess(filepath.Base(g.FilePath))
	if err != nil {
		return err
	}

	switch guess.Type {
	case guessit.Episode:
		g.Guessit = &Guessit{
			Type:         ShowType,
			ShowName:     guess.ShowName,
			Season:       fmt.Sprintf("%d", guess.Season),
			Episode:      fmt.Sprintf("%d", guess.Episode),
			Year:         fmt.Sprintf("%d", guess.Year),
			ReleaseGroup: guess.ReleaseGroup,
		}
	case guessit.Movie:
		g.Guessit = &Guessit{
			Type:         MovieType,
			MovieName:    guess.Title,
			Year:         fmt.Sprintf("%d", guess.Year),
			ReleaseGroup: guess.ReleaseGroup,
		}
	}

	return nil
}

// Guess helps find the best guess
func (g *Guesser) Guess() (*Guess, error) {
	// If nothing was found on imdb and guessit we're done
	if g.Guessit == nil && len(g.OpenSubtitle) == 0 {
		return nil, fmt.Errorf("openguessit: guessit and opensub failed to guess this file")
	}

	// If only guessit was found
	if len(g.OpenSubtitle) == 0 {
		return &Guess{Guessit: g.Guessit}, nil
	}

	// Rank the datas provided by OpenSubtitle and guessit
	guesses := []*Guess{}
	for _, sub := range g.OpenSubtitle {
		tmpGuess := &Guess{OpenSubtitle: sub}
		if g.Guessit != nil {
			if g.Guessit.Type == sub.Type {
				tmpGuess.Guessit = g.Guessit
				tmpGuess.Ranking += 10
			}
			// Movie
			if g.Guessit.Type == MovieType {
				d := levenshtein.Distance(sub.MovieName, g.Guessit.MovieName)
				// d == 0 means exact same name
				if d == 0 {
					d = 1
				}
				tmpGuess.Ranking += (1 / float64(d)) * 100
			}

			// Show
			if g.Guessit.Type == ShowType {
				if sub.Episode == g.Guessit.Episode && sub.Season == g.Guessit.Season {
					tmpGuess.Ranking += 100
				}

				d := levenshtein.Distance(sub.ShowName, g.Guessit.ShowName)
				if d == 0 {
					d = 1
				}
				tmpGuess.Ranking += (1 / float64(d)) * 100
			}

			// Year
			if sub.Year == g.Guessit.Year {
				tmpGuess.Ranking += 10
			}
		}

		// LevDist
		if sub.LevDist == 0 {
			sub.LevDist = 1
		}
		tmpGuess.Ranking += (1 / float64(sub.LevDist)) * 100

		guesses = append(guesses, tmpGuess)
	}

	var bestGuess *Guess
	for _, guess := range guesses {
		// First value
		if bestGuess == nil {
			bestGuess = guess
			continue
		}

		if guess.Ranking > bestGuess.Ranking {
			bestGuess = guess
		}
	}

	return bestGuess, nil
}

// toUpperCaseFirst is an helper to get the uppercase first of a string
func toUpperCaseFirst(s string) string {
	retStr := []string{}
	strs := strings.Split(s, " ")
	for _, str := range strs {
		if len(str) > 1 {
			str = strings.ToUpper(string(str[0])) + str[1:]
		}
		retStr = append(retStr, str)
	}

	return strings.Join(retStr, " ")
}
