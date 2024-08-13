package papi

import (
	index "github.com/odwrtw/polochon/lib/media_index"
)

// File represents a file
type File struct {
	*index.File
	resource Resource
}

// NewFile returns a new file
func NewFile(from *index.File, linkedTo Resource) *File {
	if from == nil {
		return nil
	}

	return &File{
		File:     from,
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
