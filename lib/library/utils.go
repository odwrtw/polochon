package library

import (
	"io"
	"net/http"
	"os"

	"github.com/odwrtw/polochon/lib/nfo"
)

// download helps download a file to a path
func download(URL, savePath string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Create the file
	file, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// readNFOFile reads the NFO file
func readNFOFile(filePath string, i any) error {
	// Open the file
	nfoFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = nfoFile.Close() }()

	return nfo.Read(nfoFile, i)
}

// writeNFOFile write the NFO into a file
func writeNFOFile(filePath string, i any) error {
	// Open the file
	nfoFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func() { _ = nfoFile.Close() }()

	return nfo.Write(nfoFile, i)
}

// exists is a func to check if a path exists. It could be a file or a folder,
// this function does not tell the difference
func exists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

// MoveFile is a small function that tries to rename a file, and if it fails,
// tries to manually move a file by copying + deleting it
func MoveFile(from string, to string) error {
	// First try to rename
	err := os.Rename(from, to)
	switch err.(type) {
	case nil:
		return nil
	case *os.LinkError:
		// Rename failed, and it's a LinkError, try to copy and delete the file
		source, err := os.Open(from)
		if err != nil {
			return err
		}
		defer func() { _ = source.Close() }()

		destination, err := os.Create(to)
		if err != nil {
			return err
		}
		defer func() { _ = destination.Close() }()

		_, err = io.Copy(destination, source)
		if err != nil {
			return err
		}
		return os.Remove(from)
	default:
		// If it's not a LinkError, return it
		return err
	}
}
