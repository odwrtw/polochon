package polochon

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Sirupsen/logrus"
)

func TestFilePathWithoutExt(t *testing.T) {
	for path, expected := range map[string]string{
		"/home/file.go": "/home/file",
		"file.go":       "file",
		"file":          "file",
	} {
		file := NewFile(path)
		got := file.filePathWithoutExt()

		if got != expected {
			t.Errorf("got %q, expected %q", got, expected)
		}
	}
}

func TestMovieFanartPath(t *testing.T) {
	file := NewFile("test")
	expected := "test-fanart.jpg"
	got := file.MovieFanartPath()

	if got != expected {
		t.Errorf("got %q, expected %q", got, expected)
	}
}

func TestIgnorePath(t *testing.T) {
	file := NewFile("test")
	expected := "test.ignore"
	got := file.IgnorePath()

	if got != expected {
		t.Errorf("got %q, expected %q", got, expected)
	}
}

func TestIgnoreFile(t *testing.T) {
	// Create a temp dir
	tmpDir, err := ioutil.TempDir(os.TempDir(), "polochon-file-test")
	if err != nil {
		t.Fatalf("failed to create temp dir for file tests")
	}
	defer os.RemoveAll(tmpDir)

	// Create temp file in the temp dir
	f, err := ioutil.TempFile(tmpDir, "polochon-fake-file")
	if err != nil {
		t.Fatalf("failed to create fake movie file in movie store test")
	}

	// Create the file to test
	file := NewFile(f.Name())

	// The file was just created, it should exist
	if !file.Exists() {
		t.Fatal("the file should exists")
	}

	// It should not be ignored
	if file.IsIgnored() {
		t.Fatal("the file should not be ignored yet")
	}

	// Create an ignored file
	if err := file.Ignore(); err != nil {
		t.Fatalf("failed to create ignore file: %q", err)
	}

	// It should be ignored now
	if !file.IsIgnored() {
		t.Fatal("the file should be ignored")
	}
}

func TestIsVideo(t *testing.T) {
	// Create a temp dir
	tmpDir, err := ioutil.TempDir(os.TempDir(), "polochon-file-test")
	if err != nil {
		t.Fatalf("failed to create temp dir for file tests")
	}
	defer os.RemoveAll(tmpDir)

	// Create temp file in the temp dir
	f, err := os.Create(filepath.Join(tmpDir, "fake.mp4"))
	if err != nil {
		t.Fatalf("failed to create fake file: %q", err)
	}
	f.Close()

	// Create the file to test
	config := FileConfig{
		VideoExtentions:       []string{".mp4"},
		ExcludeFileContaining: []string{"sample"},
	}
	file := NewFileWithConfig(f.Name(), config)

	if !file.IsVideo() {
		t.Fatalf("the file should be a video")
	}

	if file.IsExcluded() {
		t.Fatalf("the file should not be excluded")
	}
}

func TestExludedFile(t *testing.T) {
	// Create a temp dir
	tmpDir, err := ioutil.TempDir(os.TempDir(), "polochon-file-test")
	if err != nil {
		t.Fatalf("failed to create temp dir for file tests")
	}
	defer os.RemoveAll(tmpDir)

	// File to be created
	fileName := filepath.Join(tmpDir, "sample.avi")

	// Create the file to test
	config := FileConfig{
		ExcludeFileContaining: []string{"sample"},
		VideoExtentions:       []string{".mp4"},
	}
	file := NewFileWithConfig(fileName, config)

	if file.Exists() {
		t.Fatalf("file should not exist yet")
	}

	// Create temp file in the temp dir
	f, err := os.Create(fileName)
	if err != nil {
		t.Fatalf("failed to create fake file: %q", err)
	}
	f.Close()

	if file.IsVideo() {
		t.Fatalf("the file should not be considered as a video")
	}

	if !file.IsExcluded() {
		t.Fatalf("the file should be excluded")
	}
}

func TestDownloadImages(t *testing.T) {
	fakeLogger := logrus.NewEntry(logrus.New())

	// Http test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	// Create a fake file
	file, err := ioutil.TempFile(os.TempDir(), "polochon-image")
	if err != nil {
		t.Fatalf("failed to create fake image file")
	}
	defer os.Remove(file.Name())

	for savePath, URL := range map[string]string{
		filepath.Join(os.TempDir(), "fakeImage.jpg"): ts.URL,
		file.Name(): ts.URL,
	} {
		if err := download(URL, savePath, fakeLogger); err != nil {
			t.Errorf("failed to download image: %q", err)
		}
		defer os.Remove(savePath)

		if _, err := os.Stat(savePath); err != nil {
			t.Errorf("image file was not created")
		}
	}
}
