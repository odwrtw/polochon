package opensubtitles

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"

	polochon "github.com/odwrtw/polochon/lib"
)

const (
	moduleName = "opensubtitles"
	apiBase    = "https://api.opensubtitles.com/api/v1"
)

var _ polochon.Subtitler = (*opensubs)(nil)

var (
	ErrQuotaExceeded    = errors.New("opensubtitles: daily download quota exceeded")
	ErrInvalidVideoType = errors.New("opensubtitles: invalid video type")
)

// Params holds the YAML-configurable parameters.
type Params struct {
	APIKey   string `yaml:"api_key"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	APIBase  string `yaml:"api_base"` // optional; defaults to https://api.opensubtitles.com/api/v1
}

// opensubs implements polochon.Subtitler using the OpenSubtitles.com REST API.
type opensubs struct {
	apiKey   string
	username string
	password string
	apiBase  string

	mu        sync.Mutex
	remaining int
	resetAt   time.Time
}

func init() { polochon.RegisterModule(&opensubs{}) }

func (o *opensubs) Name() string { return moduleName }

func (o *opensubs) Init(params []byte) error {
	p := &Params{}
	if err := yaml.Unmarshal(params, p); err != nil {
		return err
	}
	return o.InitWithParams(p)
}

func (o *opensubs) InitWithParams(p *Params) error {
	o.apiKey = p.APIKey
	o.username = p.Username
	o.password = p.Password
	o.apiBase = p.APIBase
	if o.apiBase == "" {
		o.apiBase = apiBase
	}
	return nil
}

func (o *opensubs) Status() (polochon.ModuleStatus, error) {
	if o.apiKey == "" {
		return polochon.StatusFail, errors.New("opensubtitles: missing api_key")
	}
	params := url.Values{"imdb_id": {"1254207"}} // Big Buck Bunny (tt1254207)
	req, err := o.newRequest(http.MethodGet, "/subtitles?"+params.Encode(), nil)
	if err != nil {
		return polochon.StatusFail, err
	}
	resp, err := doRequest(req)
	if err != nil {
		return polochon.StatusFail, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return polochon.StatusFail, fmt.Errorf("opensubtitles: status check returned %d", resp.StatusCode)
	}
	return polochon.StatusOK, nil
}

func (o *opensubs) GetSubtitle(i any, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	entries, err := o.ListSubtitles(i, lang, log)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, polochon.ErrNoSubtitleFound
	}
	return o.DownloadSubtitle(i, entries[0], log)
}

func videoPath(i any) string {
	switch v := i.(type) {
	case *polochon.Movie:
		return v.Path
	case *polochon.ShowEpisode:
		return v.Path
	}
	return ""
}

func imdbParam(i any) string {
	switch v := i.(type) {
	case *polochon.Movie:
		return strings.TrimPrefix(v.ImdbID, "tt")
	case *polochon.ShowEpisode:
		return strings.TrimPrefix(v.ShowImdbID, "tt")
	}
	return ""
}

func titleParams(i any, lang polochon.Language) (url.Values, error) {
	p := url.Values{"languages": {lang.ShortForm()}}
	switch v := i.(type) {
	case *polochon.Movie:
		p.Set("query", v.Title)
	case *polochon.ShowEpisode:
		p.Set("query", v.ShowTitle)
		p.Set("season_number", strconv.Itoa(v.Season))
		p.Set("episode_number", strconv.Itoa(v.Episode))
	default:
		return nil, ErrInvalidVideoType
	}
	return p, nil
}

func setLang(entries []*polochon.SubtitleEntry, lang polochon.Language) []*polochon.SubtitleEntry {
	for _, e := range entries {
		e.Language = lang
	}
	return entries
}

func (o *opensubs) ListSubtitles(i any, lang polochon.Language, _ *logrus.Entry) ([]*polochon.SubtitleEntry, error) {
	// Bail early if the daily download quota is exhausted, searching is
	// pointless since we won't be able to download whatever we find.
	if o.quotaReached() {
		return nil, ErrQuotaExceeded
	}

	// Tier 1: hash search (most accurate, requires file on disk)
	if path := videoPath(i); path != "" {
		if hash, err := hashFile(path); err == nil {
			p := url.Values{"moviehash": {hash}, "languages": {lang.ShortForm()}}
			if entries, err := o.search(p); err == nil && len(entries) > 0 {
				return setLang(entries, lang), nil
			}
		}
	}

	// Tier 2: IMDB ID search
	if id := imdbParam(i); id != "" {
		p := url.Values{"imdb_id": {id}, "languages": {lang.ShortForm()}}
		if ep, ok := i.(*polochon.ShowEpisode); ok {
			p.Set("season_number", strconv.Itoa(ep.Season))
			p.Set("episode_number", strconv.Itoa(ep.Episode))
		}
		if entries, err := o.search(p); err == nil && len(entries) > 0 {
			return setLang(entries, lang), nil
		}
	}

	// Tier 3: title/query search
	params, err := titleParams(i, lang)
	if err != nil {
		return nil, err
	}
	entries, err := o.search(params)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, polochon.ErrNoSubtitleFound
	}
	return setLang(entries, lang), nil
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (o *opensubs) login() (string, error) {
	body, err := json.Marshal(loginRequest{Username: o.username, Password: o.password})
	if err != nil {
		return "", err
	}
	req, err := o.newRequest(http.MethodPost, "/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	resp, err := doRequest(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("opensubtitles: login returned status %d", resp.StatusCode)
	}
	var lr loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return "", err
	}
	if lr.Token == "" {
		return "", fmt.Errorf("opensubtitles: login returned empty token")
	}
	return lr.Token, nil
}

func (o *opensubs) logout(token string) {
	req, err := o.newRequest(http.MethodDelete, "/logout", nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := doRequest(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

type downloadRequest struct {
	FileID int `json:"file_id"`
}

type downloadResponse struct {
	Link         string `json:"link"`
	Remaining    int    `json:"remaining"`
	ResetTimeUTC string `json:"reset_time_utc"`
}

// quotaReached returns true if the daily download quota is known to be exhausted.
func (o *opensubs) quotaReached() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.remaining == 0 && !o.resetAt.IsZero() && time.Now().Before(o.resetAt)
}

// updateQuota stores the remaining count and reset time returned by the download endpoint.
// The API returns timestamps with milliseconds (e.g. "2022-04-08T13:03:16.000Z").
func (o *opensubs) updateQuota(remaining int, resetTimeUTC string) {
	t, err := time.Parse(time.RFC3339Nano, resetTimeUTC)
	if err != nil {
		return
	}
	o.mu.Lock()
	o.remaining = remaining
	o.resetAt = t
	o.mu.Unlock()
}

// DownloadSubtitle fetches the subtitle identified by entry.Token (a permanent file_id).
func (o *opensubs) DownloadSubtitle(i any, entry *polochon.SubtitleEntry, _ *logrus.Entry) (*polochon.Subtitle, error) {
	if o.quotaReached() {
		return nil, ErrQuotaExceeded
	}

	token, err := o.login()
	if err != nil {
		return nil, err
	}
	defer o.logout(token)

	fileID, err := strconv.Atoi(entry.Token)
	if err != nil {
		return nil, fmt.Errorf("opensubtitles: invalid token %q", entry.Token)
	}

	body, err := json.Marshal(downloadRequest{FileID: fileID})
	if err != nil {
		return nil, err
	}

	req, err := o.newRequest(http.MethodPost, "/download", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrQuotaExceeded
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opensubtitles: download returned status %d", resp.StatusCode)
	}

	var dr downloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		return nil, err
	}
	o.updateQuota(dr.Remaining, dr.ResetTimeUTC)

	// Fetch the actual subtitle file from the pre-signed temp URL (no auth headers needed)
	getReq, err := http.NewRequest(http.MethodGet, dr.Link, nil)
	if err != nil {
		return nil, err
	}

	fileResp, err := doRequest(getReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = fileResp.Body.Close() }()

	if fileResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opensubtitles: file fetch returned status %d", fileResp.StatusCode)
	}

	data, err := io.ReadAll(fileResp.Body)
	if err != nil {
		return nil, err
	}

	v, ok := i.(polochon.Video)
	if !ok {
		return nil, ErrInvalidVideoType
	}
	s := polochon.NewSubtitleFromVideo(v, entry.Language)
	s.Data = data
	return s, nil
}

// hashFile computes the OpenSubtitles hash for a video file.
// Algorithm: sum of first + last 64 KB as little-endian uint64 words, plus file size.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}

	const chunkSize = 65536
	if fi.Size() < chunkSize {
		return "", fmt.Errorf("opensubtitles: file too small to hash (%d bytes)", fi.Size())
	}
	hash := uint64(fi.Size())

	addChunk := func() {
		var word [8]byte
		for range chunkSize / 8 {
			if _, err := io.ReadFull(f, word[:]); err != nil {
				break
			}
			hash += binary.LittleEndian.Uint64(word[:])
		}
	}

	addChunk() // first 64 KB

	offset := max(fi.Size()-chunkSize, 0)
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return "", err
	}
	addChunk() // last 64 KB

	return fmt.Sprintf("%016x", hash), nil
}

// doRequest is overridable for tests.
var doRequest = func(req *http.Request) (*http.Response, error) {
	return http.DefaultClient.Do(req)
}

// newRequest builds an authenticated request to the OpenSubtitles REST API.
func (o *opensubs) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, o.apiBase+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Api-Key", o.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "polochon")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

type searchResponse struct {
	Data []struct {
		Attributes struct {
			Release string  `json:"release"`
			Ratings float64 `json:"ratings"`
			Files   []struct {
				FileID int `json:"file_id"`
			} `json:"files"`
		} `json:"attributes"`
	} `json:"data"`
}

func (o *opensubs) search(params url.Values) ([]*polochon.SubtitleEntry, error) {
	req, err := o.newRequest(http.MethodGet, "/subtitles?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opensubtitles: search returned status %d", resp.StatusCode)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	var entries []*polochon.SubtitleEntry
	for _, item := range sr.Data {
		if len(item.Attributes.Files) == 0 {
			continue
		}
		entries = append(entries, &polochon.SubtitleEntry{
			Release: item.Attributes.Release,
			Rating:  fmt.Sprintf("%.1f", item.Attributes.Ratings),
			Token:   strconv.Itoa(item.Attributes.Files[0].FileID),
			Source:  moduleName,
		})
	}
	return entries, nil
}
