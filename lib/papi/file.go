package papi

import (
	polochon "github.com/odwrtw/polochon/lib"
)

// File represents a downloadable sidecar file (fanart, poster, nfo, etc.)
type File struct {
	*polochon.File
	resource Resource
}

// NewFile returns a new downloadable File wrapping a polochon sidecar file.
// Returns nil if f is nil.
func NewFile(f *polochon.File, linkedTo Resource) *File {
	if f == nil {
		return nil
	}

	return &File{
		File:     f,
		resource: linkedTo,
	}
}

func (f *File) uri() (string, error) {
	if f.File == nil {
		return "", ErrMissingFile
	}

	if f.resource == nil {
		return "", ErrMissingFileResource
	}

	uri, err := f.resource.uri()
	if err != nil {
		return "", err
	}

	uri += "/files/" + f.Name
	return uri, nil
}

func (f *File) getDetails(c *Client) error {
	return ErrNotImplemented
}

func (f *File) downloadURL() (string, error) {
	uri, err := f.uri()
	if err != nil {
		return "", err
	}

	return uri, nil
}
