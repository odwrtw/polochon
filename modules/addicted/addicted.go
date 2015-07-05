package addicted

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/arbovm/levenshtein"
	"gitlab.quimbo.fr/odwrtw/addicted"
	"gitlab.quimbo.fr/odwrtw/polochon/lib"
)

var langTranslate = map[polochon.Language]string{
	polochon.EN: "english",
	polochon.FR: "french",
}

// Register a new Subtitiler
func init() {
	polochon.RegisterSubtitiler("addicted", New)
}

// New module
func New(params map[string]interface{}, log *logrus.Entry) (polochon.Subtitiler, error) {
	var user, password, lang string

	for ptr, param := range map[*string]string{
		&user:     "user",
		&password: "password",
		&lang:     "lang",
	} {
		p, ok := params[param]
		if !ok {
			continue
		}

		v, ok := p.(string)
		if !ok {
			return nil, fmt.Errorf("addicted: %s should be a string", param)
		}

		*ptr = v
	}

	if lang == "" {
		return nil, fmt.Errorf("addicted: missing lang param")
	}

	// Handle auth if the user and password are provided
	var client *addicted.Client

	var err error
	if user != "" && password != "" {
		client, err = addicted.NewWithAuth(user, password)
	} else {
		client, err = addicted.New()
	}
	if err != nil {
		return nil, err
	}

	language := polochon.Language(lang)

	// if language not available in addicted
	addictedLang, ok := langTranslate[language]
	if !ok {
		return nil, fmt.Errorf("addicted: language no supported")
	}

	return &addictedProxy{client: *client, language: addictedLang}, nil
}

type addictedProxy struct {
	client   addicted.Client
	language string
}

func (a *addictedProxy) GetShowSubtitle(reqEpisode *polochon.ShowEpisode) (polochon.Subtitle, error) {
	// TODO: add year
	// TODO: handle release

	shows, err := a.client.GetTvShows()
	if err != nil {
		return nil, err
	}
	var guessID string
	guessDist := 1000
	for showName, showID := range shows {
		dist := levenshtein.Distance(strings.ToLower(showName), strings.ToLower(reqEpisode.ShowTitle))
		if dist < guessDist {
			guessDist = dist
			guessID = showID
		}
	}

	subtitles, err := a.client.GetSubtitles(guessID, reqEpisode.Season, reqEpisode.Episode)
	if err != nil {
		return nil, err
	}

	filteredSubs := subtitles.FilterByLang(a.language)
	if len(filteredSubs) == 0 {
		return nil, polochon.ErrNoSubtitleFound
	}
	sort.Sort(addicted.ByDownloads(filteredSubs))
	return &filteredSubs[0], err
}

func (a *addictedProxy) GetMovieSubtitle(b *polochon.Movie) (polochon.Subtitle, error) {
	return nil, polochon.ErrNoSubtitleFound
}
