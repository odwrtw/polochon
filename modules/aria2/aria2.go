package aria2

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gregdel/argo/rpc"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

// Module constants
const (
	moduleName = "aria2"
)

// Register a new Downloader
func init() {
	polochon.RegisterDownloader(moduleName, NewFromRawYaml)
}

// Params represents the module params
type Params struct {
	URL    string `yaml:"url"`
	Secret string `yaml:"secret"`
}

// Client holds the connection with transmission
type Client struct {
	*Params
	protocol rpc.Protocol
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Downloader, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// New module
func New(params *Params) (polochon.Downloader, error) {
	if params.URL == "" {
		return nil, fmt.Errorf("aria2: missing URL")
	}

	if params.Secret == "" {
		return nil, fmt.Errorf("aria2: missing rpc secret")
	}

	client := &Client{}
	var err error
	client.protocol, err = rpc.New(params.URL, params.Secret)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Name implements the Module interface
func (c *Client) Name() string {
	return moduleName
}

// Status implements the Module interface
func (c *Client) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

// Download implements the downloader interface
func (c *Client) Download(URI string, log *logrus.Entry) error {
	_, err := c.protocol.AddURI(URI)
	return err
}

// List implements the downloader interface
func (c *Client) List() ([]polochon.Downloadable, error) {
	list := []rpc.StatusInfo{}

	// Get active downloads
	l, err := c.protocol.TellActive()
	if err != nil {
		return nil, err
	}
	if len(l) != 0 {
		list = append(list, l...)
	}

	// Get stopped and waiting downloads
	for _, f := range []func(int, int, ...string) ([]rpc.StatusInfo, error){
		c.protocol.TellWaiting,
		c.protocol.TellStopped,
	} {
		l, err := f(0, 10)
		if err != nil {
			return nil, err
		}

		if len(l) != 0 {
			list = append(list, l...)
		}
	}

	result := []polochon.Downloadable{}
	for _, e := range list {
		result = append(result, NewTorrentStatus(e))
	}

	return result, nil
}

// Remove implements the downloader interface
func (c *Client) Remove(d polochon.Downloadable) error {
	infos := d.Infos()
	if infos == nil {
		return fmt.Errorf("aria2: got nil downloadable")
	}

	if infos.ID == "" {
		return fmt.Errorf("aria2: no id to remove the download")
	}

	_, err := c.protocol.Remove(infos.ID)
	if err != nil {
		if strings.Contains(err.Error(), "Active Download not found") {
			// This downloadable is not active
		} else {
			return err
		}
	}

	_, err = c.protocol.PurgeDownloadResult()
	return err
}

// TorrentStatus represents the status of a torrent
type TorrentStatus struct {
	rpc.StatusInfo
}

// NewTorrentStatus creates a TorrentStatus from a rcp.StatusInfo
func NewTorrentStatus(si rpc.StatusInfo) *TorrentStatus {
	return &TorrentStatus{si}
}

// Infos implement the downloadable interface
func (ts *TorrentStatus) Infos() *polochon.DownloadableInfos {
	i := polochon.DownloadableInfos{
		ID:   ts.StatusInfo.Gid,
		Name: ts.StatusInfo.BitTorrent.Info.Name,
	}

	// Add the filePaths
	i.FilePaths = []string{}
	for _, f := range ts.StatusInfo.Files {
		i.FilePaths = append(i.FilePaths, f.Path)
	}

	// Set the path as the default name
	if i.Name == "" && len(i.FilePaths) > 0 {
		i.Name = i.FilePaths[0]
	}

	for i, s := range map[*int]string{
		&i.DownloadRate:   ts.StatusInfo.DownloadSpeed,
		&i.UploadRate:     ts.StatusInfo.UploadSpeed,
		&i.DownloadedSize: ts.StatusInfo.CompletedLength,
		&i.UploadedSize:   ts.StatusInfo.UploadLength,
		&i.TotalSize:      ts.StatusInfo.TotalLength,
	} {
		var err error
		*i, err = strconv.Atoi(s)
		if err != nil {
			return nil
		}
	}

	if ts.StatusInfo.CompletedLength == ts.StatusInfo.TotalLength {
		i.IsFinished = true
		i.PercentDone = 100
	} else {
		i.PercentDone = float32(i.DownloadedSize) * 100 / float32(i.TotalSize)
	}

	if i.UploadedSize != 0 {
		i.Ratio = float32(i.UploadedSize) / float32(i.TotalSize)
	}

	return &i
}
