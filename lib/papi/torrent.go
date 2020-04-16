package papi

import (
	"fmt"
	"sort"

	polochon "github.com/odwrtw/polochon/lib"
)

// AddTorrent will tell polochon to add this torrent URL
func (c *Client) AddTorrent(torrent *polochon.Torrent) error {
	if torrent == nil {
		return ErrMissingTorrentData
	}

	if torrent.Result == nil || torrent.Result.URL == "" {
		return ErrMissingTorrentURL
	}

	result := struct {
		Message string `json:"message"`
	}{}

	return c.post(c.endpoint+"/torrents", torrent, result)
}

// GetTorrents will tell polochon to list its torrents
func (c *Client) GetTorrents() ([]*polochon.Torrent, error) {
	torrents := []*polochon.Torrent{}
	url := fmt.Sprintf("%s/%s", c.endpoint, "torrents")

	if err := c.get(url, &torrents); err != nil {
		return nil, err
	}

	sort.Slice(torrents, func(i, j int) bool {
		return torrents[i].Status.ID < torrents[j].Status.ID
	})

	return torrents, nil
}

// RemoveTorrent will tell polochon to remove this torrent
func (c *Client) RemoveTorrent(ID string) error {
	url := fmt.Sprintf("%s/%s/%s", c.endpoint, "torrents", ID)

	return c.delete(url)
}
