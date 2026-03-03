package podnapisi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/agnivade/levenshtein"
)

var baseURL = "https://www.podnapisi.net"

var (
	ErrNotAVideo        = errors.New("podnapisi: not a video")
	ErrNoPublishID      = errors.New("podnapisi: subtitle has no publish_id")
	ErrNoSRTInArchive   = errors.New("podnapisi: no .srt file found in archive")
	ErrUnexpectedStatus = errors.New("podnapisi: unexpected API status")
	ErrTooManyRequests  = errors.New("podnapisi: too many requests")
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Overridable for tests.
var (
	podnapisiSearch   = search
	podnapisiDownload = downloadBest
)

type subtitleMovie struct {
	Providers []string `json:"providers"`
	Year      int      `json:"year"`
	Title     string   `json:"title"`
}

type subtitle struct {
	PublishID      string        `json:"publish_id"`
	Language       string        `json:"language"`
	CustomReleases []string      `json:"custom_releases"`
	Flags          []string      `json:"flags"`
	Movie          subtitleMovie `json:"movie"`
}

type searchResponse struct {
	Data     []*subtitle `json:"data"`
	AllPages int         `json:"all_pages"`
	Status   string      `json:"status"`
}

// search queries the Podnapisi JSON API with the given parameters.
func search(params url.Values) ([]*subtitle, error) {
	searchURL := baseURL + "/subtitles/search/advanced?" + params.Encode()
	req, err := http.NewRequest(http.MethodGet, searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrTooManyRequests
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("podnapisi: search returned HTTP %d", resp.StatusCode)
	}

	var sr searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, err
	}

	switch sr.Status {
	case "", "ok":
		// success
	case "too-many-requests":
		return nil, ErrTooManyRequests
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnexpectedStatus, sr.Status)
	}

	return sr.Data, nil
}

// downloadBest downloads the zip archive for publishID and returns the bytes
// of the .srt file whose name is closest to releaseName.
func downloadBest(publishID, releaseName string) ([]byte, error) {
	if publishID == "" {
		return nil, ErrNoPublishID
	}

	downloadURL := baseURL + "/subtitles/" + publishID + "/download"
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrTooManyRequests
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("podnapisi: download returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return extractBestSRT(body, releaseName)
}

// extractBestSRT opens zipData as a zip archive and returns the bytes of the
// .srt file whose filename is closest to releaseName via Levenshtein distance.
func extractBestSRT(zipData []byte, releaseName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, err
	}

	var bestFile *zip.File
	bestScore := -1

	for _, f := range zr.File {
		if !strings.HasSuffix(strings.ToLower(f.Name), ".srt") {
			continue
		}
		name := strings.TrimSuffix(f.Name, ".srt")
		score := levenshtein.ComputeDistance(releaseName, name)
		if bestFile == nil || score < bestScore {
			bestFile = f
			bestScore = score
		}
	}

	if bestFile == nil {
		return nil, ErrNoSRTInArchive
	}

	rc, err := bestFile.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()

	return io.ReadAll(rc)
}
