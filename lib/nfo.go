package polochon

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/Sirupsen/logrus"
)

// readNFO deserialized a XML file from a reader
func readNFO(r io.Reader, i interface{}) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	err = xml.Unmarshal(b, i)
	if err != nil {
		return err
	}

	return nil
}

// writeNFO serialized a XML into writer
func writeNFO(w io.Writer, i interface{}) error {
	b, err := xml.MarshalIndent(i, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	if err != nil {
		return err
	}

	return nil
}

// MarshalInFile write a nfo into a file
func MarshalInFile(i interface{}, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the data into the file
	return writeNFO(file, i)
}

// download is an helper to download a file from its URL
func download(URL, savePath string, log *logrus.Entry) error {
	// Check if the file as already been downladed
	if _, err := os.Stat(savePath); err == nil {
		log.Debugf("File already downladed : %q", savePath)
		return nil
	}

	log.Debugf("Downloading file %q into %q", URL, savePath)

	// Download
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
