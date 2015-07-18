package opensubtitles

import (
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
	"github.com/oz/osdb"
)

// Struct of a subtitle containing an osdbSubtitle, a connexion if any, and a
// link to the osdb.Client
type openSubtitle struct {
	os     *osdb.Subtitle
	conn   io.ReadCloser
	client *osdb.Client
}

var langTranslate = map[polochon.Language]string{
	polochon.EN: "eng",
	polochon.FR: "fre",
}

// Register a new Subtitler
func init() {
	polochon.RegisterSubtitler("opensubtitles", New)
}

// Close the subtitle connexion
func (o *openSubtitle) Close() {
	if o.conn != nil {
		o.conn.Close()
	}
}

// Read the subtitle
func (o *openSubtitle) Read(b []byte) (int, error) {
	id, err := strconv.Atoi(o.os.IDSubtitleFile)
	if err != nil {
		return 0, err
	}

	// Download
	if o.conn == nil {
		files, err := o.client.DownloadSubtitles([]int{id})
		if err != nil {
			return 0, err
		}
		if len(files) == 0 {
			return 0, fmt.Errorf("No file match this subtitle ID")
		}

		// Save to disk.
		r, err := files[0].Reader()
		if err != nil {
			return 0, err
		}
		o.conn = r
	}

	return o.conn.Read(b)
}

// New module
func New(params map[string]interface{}, log *logrus.Entry) (polochon.Subtitler, error) {
	// Get all the needed params
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
			return nil, fmt.Errorf("opensubtitles: %s should be a string", param)
		}

		*ptr = v
	}

	if user == "" {
		log.Debugf("Logging in opensubtitles with no user")
	}
	if user != "" && password == "" {
		return nil, fmt.Errorf("opensubtitles: missing password param")
	}
	if lang == "" {
		return nil, fmt.Errorf("opensubtitles: missing lang param")
	}

	language := polochon.Language(lang)

	opensubtitlesLang, ok := langTranslate[language]
	if !ok {
		return nil, fmt.Errorf("opensubtitles: language no supported")
	}

	// Create the OpenSubtitles proxy
	osp := &osProxy{
		language: opensubtitlesLang,
		user:     user,
		password: password,
		log:      log,
	}

	// Set the OpenSubtitles client
	err := osp.getOpenSubtitleClient()
	if err != nil {
		return nil, err
	}

	return osp, nil
}

// getOpenSubtitleClient will return a configured osdb.Client
func (osp *osProxy) getOpenSubtitleClient() error {
	// Create a new client if needed
	if osp.client == nil {
		client, err := osdb.NewClient()
		if err != nil {
			return err
		}
		osp.client = client
	}

	// Test to see if the connexion is still ok
	if err := osp.client.Noop(); err == nil {
		return nil
	}

	// If we had an error, try to login again
	// LogIn with the user's configuration
	return osp.client.LogIn(osp.user, osp.password, osp.language)
}

type osProxy struct {
	client   *osdb.Client
	log      *logrus.Entry
	language string
	user     string
	password string
}

func (osp *osProxy) checkSubtitles(i interface{}, subs osdb.Subtitles) (*osdb.Subtitle, error) {
	var goodSubs []osdb.Subtitle

	switch v := i.(type) {
	case polochon.ShowEpisode:
		goodSubs = osp.getGoodShowEpisodeSubtitles(v, subs)
	case polochon.Movie:
		goodSubs = osp.getGoodMovieSubtitles(v, subs)
	default:
		return nil, fmt.Errorf("Error while checking subtitles, invalid type %t", v)
	}
	return osp.getBestSubtitle(goodSubs), nil
}

// GetShowSubtitle will get a show subtitle
func (osp *osProxy) GetShowSubtitle(s *polochon.ShowEpisode) (polochon.Subtitle, error) {
	sub, err := osp.searchSubtitles(*s, s.File.Path)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// GetMovieSubtitle will get a movie subtitle
func (osp *osProxy) GetMovieSubtitle(m *polochon.Movie) (polochon.Subtitle, error) {
	sub, err := osp.searchSubtitles(*m, m.File.Path)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// searchSubtitles will search via hash, then filename, then info and return the best subtitle
func (osp *osProxy) searchSubtitles(v interface{}, filePath string) (*openSubtitle, error) {
	// Look for subtitles with the hash
	sub, err := osp.checkConnAndExec(osp.searchSubtitlesByHash, v, filePath)
	if err != nil {
		osp.log.Warnf("Got error looking for subtitle by hash : %q", err)
	}

	if sub != nil {
		osp.log.Debug("We got the subtitle by hash")
		return &openSubtitle{os: sub, client: osp.client}, nil
	}

	osp.log.Debug("Nothing in the result, need to check again with filename")

	// Look for subtitles with the filename
	sub, err = osp.checkConnAndExec(osp.searchSubtitlesByFilename, v, filePath)
	if err != nil {
		osp.log.Warnf("Got error looking for subtitle by filename : %q", err)
	}

	if sub != nil {
		osp.log.Debug("We got the subtitle by filename")
		return &openSubtitle{os: sub, client: osp.client}, nil
	}

	osp.log.Debug("Still no good, need to check again with imdbID")

	// Look for subtitles with the title and episode and season or by imdbID
	sub, err = osp.checkConnAndExec(osp.searchSubtitlesByInfos, v, filePath)
	if err != nil {
		osp.log.Warnf("Got error looking for subtitle by infos : %q", err)
	}

	if sub != nil {
		return &openSubtitle{os: sub, client: osp.client}, nil
	}
	return nil, nil
}

// checkConnAndExec will check the connexion, execute the function and check the subtitles returned
func (osp *osProxy) checkConnAndExec(f func(v interface{}, filePath string) (osdb.Subtitles, error), v interface{}, filePath string) (*osdb.Subtitle, error) {
	// Check the opensubtitle client
	err := osp.getOpenSubtitleClient()
	if err != nil {
		return nil, err
	}
	res, err := f(v, filePath)
	if err != nil {
		return nil, err
	}
	// Now that we have a list of subtitles, need to check that we have the good one
	sub, err := osp.checkSubtitles(v, res)
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// searchSubtitlesByHash will make a hash of the file, and check for corresponding subtitles
func (osp *osProxy) searchSubtitlesByHash(v interface{}, filePath string) (osdb.Subtitles, error) {
	// Set the languages
	languages := []string{osp.language}
	// Hash movie file, and search...
	res, err := osp.client.FileSearch(filePath, languages)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// searchSubtitlesByFilename will search for subtitles corresponding to the name of the file
func (osp *osProxy) searchSubtitlesByFilename(v interface{}, filePath string) (osdb.Subtitles, error) {
	var innerParams = []map[string]string{
		{
			"query":         path.Base(filePath),
			"sublanguageid": osp.language,
		},
	}

	params := []interface{}{
		osp.client.Token,
		innerParams,
	}

	res, err := osp.client.SearchSubtitles(&params)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// searchSubtitlesByInfos will take the info of the video (imdbId / title / ...) to get subtitles
func (osp *osProxy) searchSubtitlesByInfos(m interface{}, filePath string) (osdb.Subtitles, error) {

	innerParams, err := osp.openSubtitleParams(m)
	if err != nil {
		return nil, err
	}

	params := []interface{}{
		osp.client.Token,
		innerParams,
	}

	res, err := osp.client.SearchSubtitles(&params)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// openSubtitleParams will return the good params needed for a search
func (osp *osProxy) openSubtitleParams(i interface{}) ([]map[string]string, error) {
	switch v := i.(type) {
	case polochon.ShowEpisode:
		return osp.openSubtitleShowEpisodeParams(v), nil
	case polochon.Movie:
		return osp.openSubtitleMovieParams(v), nil
	default:
		return []map[string]string{}, fmt.Errorf("Not a showEpisode, not a movie, something's fucked up")
	}
}

// openSubtitleMovieParam will return the needed params to look for a movie
func (osp *osProxy) openSubtitleMovieParams(m polochon.Movie) []map[string]string {
	return []map[string]string{
		{
			"sublanguageid": osp.language,
			"imdbid":        strings.Replace(m.ImdbID, "tt", "", -1),
		},
	}
}

// openSubtitleShowEpisode will return the needed params to look for a show episode
func (osp *osProxy) openSubtitleShowEpisodeParams(s polochon.ShowEpisode) []map[string]string {
	return []map[string]string{
		{
			"query":         s.ShowTitle,
			"season":        strconv.Itoa(s.Season),
			"episode":       strconv.Itoa(s.Episode),
			"sublanguageid": osp.language,
		},
	}
}

// getBestSubtitle will get the best subtitle from the list
// Given the nb of downloads and the rating
func (osp *osProxy) getBestSubtitle(subs []osdb.Subtitle) *osdb.Subtitle {
	if len(subs) > 0 {
		return &subs[0]
	}
	return nil
}

// getGoodMovieSubtitles will retrieve only the movies with the same imdbId
func (osp *osProxy) getGoodMovieSubtitles(m polochon.Movie, subs osdb.Subtitles) []osdb.Subtitle {
	var goodSubs []osdb.Subtitle
	for _, sub := range subs {
		// Need to check that it's the good subtitle
		imdbID := fmt.Sprintf("tt%07s", sub.IDMovieImdb)

		if imdbID == m.ImdbID {
			osp.log.Debugf("This is it : %s", imdbID)
			goodSubs = append(goodSubs, sub)
		} else {
			continue
		}
	}
	return goodSubs
}

// getGoodShowEpisodeSubtitles will retrieve only the shoes with the same
// imdbId / season nb / episode nb
func (osp *osProxy) getGoodShowEpisodeSubtitles(s polochon.ShowEpisode, subs osdb.Subtitles) []osdb.Subtitle {
	var goodSubs []osdb.Subtitle
	for _, sub := range subs {
		// Need to check that it's the good subtitle
		imdbID := fmt.Sprintf("tt%07s", sub.SeriesIMDBParent)
		if imdbID != s.ShowImdbID {
			continue
		}

		if sub.SeriesEpisode != strconv.Itoa(s.Episode) {
			continue
		}

		if sub.SeriesSeason != strconv.Itoa(s.Season) {
			continue
		}

		osp.log.Debugf("This is it : %s | episode %s | season %s", imdbID, sub.SeriesEpisode, sub.SeriesSeason)
		goodSubs = append(goodSubs, sub)
	}
	return goodSubs
}
