package bsplayer

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"text/template"

	polochon "github.com/odwrtw/polochon/lib"
)

var (
	domainIndex  atomic.Uint64
	baseURL      = "http://s%d.api.bsplayer-subtitles.com/v1.php"
	domains      = []int{1, 2, 3, 4, 5, 6, 7, 8, 101, 102, 103, 104, 105, 106, 107, 108, 109}
	soapTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:SOAP-ENC="http://schemas.xmlsoap.org/soap/encoding/"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xmlns:xsd="http://www.w3.org/2001/XMLSchema"
    xmlns:ns1="{{.Endpoint}}">
    <SOAP-ENV:Body SOAP-ENV:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
        <ns1:{{.Action}}>{{.Params}}</ns1:{{.Action}}>
    </SOAP-ENV:Body>
</SOAP-ENV:Envelope>
`
	soap         = template.Must(template.New("soap").Parse(soapTemplate))
	loginPayload = `
        <username></username>
        <password></password>
        <AppID>BSPlayer v2.72</AppID>
	`
)

func getEndpoint() string {
	idx := domainIndex.Add(1)
	domain := domains[idx%uint64(len(domains))]
	return fmt.Sprintf(baseURL, domain)
}

type soapParams struct {
	Endpoint string
	Action   string
	Params   string
}

type loginResponse struct {
	Status int    `xml:"Body>logInResponse>return>result"`
	Token  string `xml:"Body>logInResponse>return>data"`
}

type subtitle struct {
	Lang   string `xml:"subLang"`
	Name   string `xml:"subName"`
	Format string `xml:"subFormat"`
	URL    string `xml:"subDownloadLink"`
	Rating string `xml:"subRating"`

	FileHash string `xml:"movieHash"`
	FileSize string `xml:"movieSize"`
	ImdbID   string `xml:"movieIMBDID"`
}

func (s *subtitle) String() string {
	var out strings.Builder
	fmt.Fprintf(&out, "Name:%s\n", s.Name)
	fmt.Fprintf(&out, "ImdbID:%s\n", s.ImdbID)
	fmt.Fprintf(&out, "Lang:%s\n", s.Lang)
	fmt.Fprintf(&out, "Format:%s\n", s.Format)
	fmt.Fprintf(&out, "Rating:%s\n", s.Rating)
	fmt.Fprintf(&out, "Hash:%s\n", s.FileHash)
	fmt.Fprintf(&out, "Size:%s\n", s.FileSize)
	fmt.Fprintf(&out, "URL:%s\n", s.URL)
	return out.String()
}

type searchResponse struct {
	Status int         `xml:"Body>searchSubtitlesResponse>return>result>result"`
	Subs   []*subtitle `xml:"Body>searchSubtitlesResponse>return>data>item"`
}

func query(endpoint, action, payload string) ([]byte, error) {
	params := &bytes.Buffer{}
	err := soap.Execute(params, &soapParams{
		Endpoint: endpoint,
		Action:   action,
		Params:   payload,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", endpoint, params)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", "BSPlayer/2.x (1022.12362)")
	req.Header.Add("Content-Type", "text/xml; charset=utf-8")
	req.Header.Add("Connection", "close")
	req.Header.Add("SOAPAction", fmt.Sprintf(`"%s#%s"`, endpoint, action))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bsplayer: http response code %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func tryLogin(endpoint string) (string, error) {
	body, err := query(endpoint, "logIn", loginPayload)
	if err != nil {
		return "", err
	}

	data := loginResponse{}
	if err := xml.Unmarshal(body, &data); err != nil {
		return "", err
	}

	if data.Status != http.StatusOK {
		return "", fmt.Errorf("bsplayer: xml login response code %d", data.Status)
	}

	if data.Token == "" {
		return "", fmt.Errorf("bsplayer: missing login token")
	}

	return data.Token, nil
}

func login() (string, string, error) {
	var err error
	for range 3 {
		endpoint := getEndpoint()
		token, e := tryLogin(endpoint)
		if e == nil {
			return endpoint, token, nil
		}
		err = e
	}
	return "", "", err
}

type queryParams struct {
	imdbID string
	size   string
	hash   string
	lang   string
}

var langs = map[polochon.Language]string{
	polochon.EN: "eng",
	polochon.FR: "fre",
}

func newQuery(imdbID string, lang polochon.Language, file *polochon.File) (*queryParams, error) {
	if file == nil {
		return nil, fmt.Errorf("bsplayer: missing file")
	}

	l, ok := langs[lang]
	if !ok {
		return nil, fmt.Errorf("bsplayer: lang %s not handled", lang)
	}

	hash, err := file.OpensubHash()
	if err != nil {
		return nil, err
	}

	return &queryParams{
		imdbID: strings.ReplaceAll(imdbID, "tt", ""),
		hash:   fmt.Sprintf("%016x", hash),
		size:   strconv.Itoa(int(file.Size)),
		lang:   l,
	}, nil
}

func search(qp *queryParams) ([]*subtitle, error) {
	endpoint, token, err := login()
	if err != nil {
		return nil, err
	}

	params := strings.Builder{}
	fmt.Fprintf(&params, "<handle>%s</handle>", token)
	fmt.Fprintf(&params, "<movieHash>%s</movieHash>", qp.hash)
	fmt.Fprintf(&params, "<movieSize>%s</movieSize>", qp.size)
	fmt.Fprintf(&params, "<languageId>%s</languageId>", qp.lang)
	fmt.Fprintf(&params, "<imdbId>%s</imdbId>", qp.imdbID)

	ret, err := query(endpoint, "searchSubtitles", params.String())
	if err != nil {
		return nil, err
	}

	data := searchResponse{}
	if err := xml.Unmarshal(ret, &data); err != nil {
		return nil, err
	}

	if data.Status != http.StatusOK {
		return nil, fmt.Errorf("bsplayer: search response code %d", data.Status)
	}

	return data.Subs, nil
}

func fetch(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bsplayer: fetch http status: %s", resp.Status)
	}

	return gzip.NewReader(resp.Body)
}
