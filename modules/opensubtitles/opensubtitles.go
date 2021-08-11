package opensubtitles

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/oz/osdb"
	"github.com/sirupsen/logrus"
)

// Make sure that the module is a subtitler
var _ polochon.Subtitler = (*osProxy)(nil)

func init() {
	polochon.RegisterModule(&osProxy{})
}

// Module constants
const (
	moduleName = "opensubtitles"
)

var langTranslate = map[polochon.Language]string{
	polochon.EN: "eng",
	polochon.FR: "fre",
}

// Opensubtitles errors
var (
	ErrInvalidArgument = errors.New("opensubtitles: invalid argument")
	ErrMissingArgument = errors.New("opensubtitles: missing argument")
)

// Params represents the module params
type Params struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Lang     string `yaml:"lang"`
}

// IsValid checks if the given params are valid
func (p *Params) IsValid() bool {
	if p.User == "" || p.Password == "" {
		return false
	}
	// Set english as the default language
	if p.Lang == "" {
		p.Lang = string(polochon.EN)
	}
	return true
}

type osProxy struct {
	client     *osdb.Client
	language   string
	user       string
	password   string
	configured bool
}

// Init implements the module interface
func (osp *osProxy) Init(p []byte) error {
	if osp.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return osp.InitWithParams(params)
}

// InitWithParams configures the module
func (osp *osProxy) InitWithParams(params *Params) error {
	if !params.IsValid() {
		return ErrMissingArgument
	}

	language := polochon.Language(params.Lang)
	opensubtitlesLang, ok := langTranslate[language]
	if !ok {
		return ErrInvalidArgument
	}

	// Create the OpenSubtitles proxy
	osp.language = opensubtitlesLang
	osp.user = params.User
	osp.password = params.Password
	osp.configured = true

	return nil
}

// Name implements the Module interface
func (osp *osProxy) Name() string {
	return moduleName
}

// Status implements the Module interface
func (osp *osProxy) Status() (polochon.ModuleStatus, error) {
	// Create a new client if needed
	if osp.client == nil {
		client, err := newOsdbClient()
		if err != nil {
			return polochon.StatusFail, err
		}
		osp.client = client
	}

	innerParams := []map[string]string{
		{
			"imdbid": "0133093",
		},
	}

	params := []interface{}{
		osp.client.Token,
		innerParams,
	}

	_, err := searchOsdbSubtitles(osp.client, params)
	if err != nil {
		return polochon.StatusFail, err
	}
	return polochon.StatusOK, nil
}

// Function to get a new client
var newOsdbClient = func() (*osdb.Client, error) {
	return osdb.NewClient()
}

// Function to check the client
var checkOsdbClient = func(c *osdb.Client) error {
	return c.Noop()
}

// Function to log in the client
var logInOsdbClient = func(c *osdb.Client, user, password, language string) error {
	return c.LogIn(user, password, language)
}

// Function to search subtitles via params
var searchOsdbSubtitles = func(c *osdb.Client, params []interface{}) (osdb.Subtitles, error) {
	return c.SearchSubtitles(&params)
}

// Function to search subtitles via a file
var fileSearchSubtitles = func(c *osdb.Client, filePath string, languages []string) (osdb.Subtitles, error) {
	return c.FileSearch(filePath, languages)
}

// getOpenSubtitleClient will return a configured osdb.Client
func (osp *osProxy) getOpenSubtitleClient() error {
	// Create a new client if needed
	if osp.client == nil {
		client, err := newOsdbClient()
		if err != nil {
			return err
		}
		osp.client = client
	}

	// Test to see if the connexion is still ok
	if err := checkOsdbClient(osp.client); err == nil {
		return nil
	}

	// If we had an error, try to login again
	// LogIn with the user's configuration
	return logInOsdbClient(osp.client, osp.user, osp.password, osp.language)
}

func (osp *osProxy) checkSubtitles(i interface{}, subs osdb.Subtitles, log *logrus.Entry) (*osdb.Subtitle, error) {
	var goodSubs []osdb.Subtitle

	switch v := i.(type) {
	case *polochon.ShowEpisode:
		goodSubs = osp.getGoodShowEpisodeSubtitles(v, subs, log)
	case *polochon.Movie:
		goodSubs = osp.getGoodMovieSubtitles(v, subs, log)
	default:
		return nil, fmt.Errorf("opensubtitles: error while checking subtitles, invalid type %s", reflect.TypeOf(v))
	}
	return osp.getBestSubtitle(goodSubs), nil
}

// searchSubtitles will search via hash, then filename, then info and return the best subtitle
func (osp *osProxy) searchSubtitles(v polochon.Video, lang string, log *logrus.Entry) (*openSubtitle, error) {
	// Look for subtitles with the hash
	sub, err := osp.checkConnAndExec(osp.searchSubtitlesByHash, v, lang, v.GetFile().Path, log)
	if err != nil {
		log.Warnf("Got error looking for subtitle by hash : %q", err)
	}

	if sub != nil {
		log.Debug("We got the subtitle by hash")
		return &openSubtitle{os: sub, client: osp.client}, nil
	}

	log.Debug("Nothing in the result, need to check again with filename")

	// Look for subtitles with the filename
	sub, err = osp.checkConnAndExec(osp.searchSubtitlesByFilename, v, lang, v.GetFile().Path, log)
	if err != nil {
		log.Warnf("Got error looking for subtitle by filename : %q", err)
	}

	if sub != nil {
		log.Debug("We got the subtitle by filename")
		return &openSubtitle{os: sub, client: osp.client}, nil
	}

	log.Debug("Still no good, need to check again with imdbID")

	// Look for subtitles with the title and episode and season or by imdbID
	sub, err = osp.checkConnAndExec(osp.searchSubtitlesByInfos, v, lang, v.GetFile().Path, log)
	if err != nil {
		log.Warnf("Got error looking for subtitle by infos : %q", err)
	}

	if sub != nil {
		return &openSubtitle{os: sub, client: osp.client}, nil
	}

	return nil, polochon.ErrNoSubtitleFound
}

// checkConnAndExec will check the connexion, execute the function and check the subtitles returned
func (osp *osProxy) checkConnAndExec(
	f func(v polochon.Video, lang string, filePath string) (osdb.Subtitles, error),
	v polochon.Video, lang string, filePath string, log *logrus.Entry) (*osdb.Subtitle, error) {

	// Check the opensubtitle client
	err := osp.getOpenSubtitleClient()
	if err != nil {
		return nil, err
	}
	res, err := f(v, lang, filePath)
	if err != nil {
		return nil, err
	}
	// Now that we have a list of subtitles, need to check that we have the good one
	return osp.checkSubtitles(v, res, log)
}

// searchSubtitlesByHash will make a hash of the file, and check for corresponding subtitles
func (osp *osProxy) searchSubtitlesByHash(_ polochon.Video, lang string, filePath string) (osdb.Subtitles, error) {
	// Set the languages
	languages := []string{lang}
	// Hash movie file, and search...
	return fileSearchSubtitles(osp.client, filePath, languages)
}

// searchSubtitlesByFilename will search for subtitles corresponding to the name of the file
func (osp *osProxy) searchSubtitlesByFilename(_ polochon.Video, lang string, filePath string) (osdb.Subtitles, error) {
	var innerParams = []map[string]string{
		{
			"query":         path.Base(filePath),
			"sublanguageid": lang,
		},
	}

	params := []interface{}{
		osp.client.Token,
		innerParams,
	}

	return searchOsdbSubtitles(osp.client, params)
}

// searchSubtitlesByInfos will take the info of the video (imdbId / title / ...) to get subtitles
func (osp *osProxy) searchSubtitlesByInfos(v polochon.Video, lang string, filePath string) (osdb.Subtitles, error) {

	innerParams, err := osp.openSubtitleParams(v)
	if err != nil {
		return nil, err
	}
	innerParams[0]["sublanguageid"] = lang

	params := []interface{}{
		osp.client.Token,
		innerParams,
	}

	return searchOsdbSubtitles(osp.client, params)
}

// openSubtitleParams will return the good params needed for a search
func (osp *osProxy) openSubtitleParams(video polochon.Video) ([]map[string]string, error) {
	switch v := video.(type) {
	case *polochon.ShowEpisode:
		return osp.openSubtitleShowEpisodeParams(v), nil
	case *polochon.Movie:
		return osp.openSubtitleMovieParams(v), nil
	default:
		return []map[string]string{}, fmt.Errorf("opensubtitles: not a showEpisode, not a movie, something's fucked up")
	}
}

// openSubtitleMovieParam will return the needed params to look for a movie
func (osp *osProxy) openSubtitleMovieParams(m *polochon.Movie) []map[string]string {
	return []map[string]string{
		{
			"imdbid": strings.Replace(m.ImdbID, "tt", "", -1),
		},
	}
}

// openSubtitleShowEpisode will return the needed params to look for a show episode
func (osp *osProxy) openSubtitleShowEpisodeParams(s *polochon.ShowEpisode) []map[string]string {
	return []map[string]string{
		{
			"query":   s.ShowTitle,
			"season":  strconv.Itoa(s.Season),
			"episode": strconv.Itoa(s.Episode),
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
func (osp *osProxy) getGoodMovieSubtitles(m *polochon.Movie, subs osdb.Subtitles, log *logrus.Entry) []osdb.Subtitle {
	var goodSubs []osdb.Subtitle
	for _, sub := range subs {
		// Need to check that it's the good subtitle
		imdbID := fmt.Sprintf("tt%07s", sub.IDMovieImdb)

		if imdbID == m.ImdbID {
			goodSubs = append(goodSubs, sub)
		} else {
			continue
		}
	}
	if len(goodSubs) > 0 {
		log.Debugf("Got %d subtitles", len(goodSubs))
	}
	return goodSubs
}

// getGoodShowEpisodeSubtitles will retrieve only the shoes with the same
// imdbId / season nb / episode nb
func (osp *osProxy) getGoodShowEpisodeSubtitles(s *polochon.ShowEpisode, subs osdb.Subtitles, log *logrus.Entry) []osdb.Subtitle {
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

		goodSubs = append(goodSubs, sub)
	}
	if len(goodSubs) > 0 {
		log.Debugf("Got %d subtitles", len(goodSubs))
	}
	return goodSubs
}

func (osp *osProxy) GetSubtitle(i interface{}, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	opensubtitlesLang, ok := langTranslate[lang]
	if !ok {
		return nil, ErrInvalidArgument
	}

	video, ok := i.(polochon.Video)
	if !ok {
		return nil, fmt.Errorf("opensub: invalid argument")
	}

	sub, err := osp.searchSubtitles(video, opensubtitlesLang, log)
	if err != nil {
		return nil, err
	}

	data := &bytes.Buffer{}
	_, err = data.ReadFrom(sub)
	if err != nil {
		return nil, err
	}

	s := polochon.NewSubtitleFromVideo(video, lang)
	s.Data = data.Bytes()
	return s, nil
}

// Struct of a subtitle containing an osdbSubtitle, a connexion if any, and a
// link to the osdb.Client
type openSubtitle struct {
	os     *osdb.Subtitle
	conn   io.ReadCloser
	client *osdb.Client
}

// Close the subtitle connexion
func (o *openSubtitle) Close() error {
	if o.conn != nil {
		return o.conn.Close()
	}
	return nil
}

// Read the subtitle
func (o *openSubtitle) Read(b []byte) (int, error) {
	// Download
	if o.conn == nil {
		files, err := o.client.DownloadSubtitles([]osdb.Subtitle{*o.os})
		if err != nil {
			return 0, err
		}
		if len(files) == 0 {
			return 0, fmt.Errorf("opensubtitles: no file match this subtitle ID")
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
