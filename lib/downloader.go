package polochon

// Downloader represent a interface for any downloader
type Downloader interface {
	Module
	Download(*Torrent) error
	Remove(*Torrent) error
	List() ([]*Torrent, error)
}
