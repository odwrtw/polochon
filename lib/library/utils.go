package library

import (
	"io"
	"net/http"
	"os"

	"github.com/odwrtw/polochon/lib"
)

// download helps download a file to a path
func download(URL, savePath string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	file, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write from the net to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// readNFOFile reads the NFO file
func readNFOFile(filePath string, i interface{}) error {
	// Open the file
	nfoFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer nfoFile.Close()

	return polochon.ReadNFO(nfoFile, i)
}

// writeNFOFile write the NFO into a file
func writeNFOFile(filePath string, i interface{}) error {
	// Open the file
	nfoFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer nfoFile.Close()

	return polochon.WriteNFO(nfoFile, i)
}

// exists is a func to check if a path exists. It could be a file or a folder,
// this function does not tell the difference
func exists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
