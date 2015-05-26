package polochon

// Downloader represent a interface for any downloader
type Downloader interface {
	Download(URL string) error
	List() error
}

// Downloadable is an interface for anything to be downlaoded
type Downloadable interface {
	Status() error
	Remove() error
	Download() error
}
