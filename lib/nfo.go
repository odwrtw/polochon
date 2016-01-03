package polochon

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"os"
)

// ShowNFO represents a show NFO in kodi
type ShowNFO struct {
	Title     string  `xml:"title"`
	ShowTitle string  `xml:"showtitle"`
	Rating    float32 `xml:"rating"`
	Plot      string  `xml:"plot"`
	URL       string  `xml:"episodeguide>url"`
	TvdbID    int     `xml:"tvdbid"`
	ImdbID    string  `xml:"imdbid"`
	Year      int     `xml:"year"`
	Premiered string  `xml:"premiered"`
}

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
var MarshalInFile = func(i interface{}, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the data into the file
	return writeNFO(file, i)
}
