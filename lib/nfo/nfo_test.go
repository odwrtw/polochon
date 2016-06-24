package nfo

import (
	"bytes"
	"testing"
)

func TestReadInvalidType(t *testing.T) {
	buffer := &bytes.Buffer{}
	var invalidType int

	if err := Read(buffer, invalidType); err == nil {
		t.Fatalf("got no error, expected: %q", ErrInvalidType)
	}
}

func TestWriteInvalidType(t *testing.T) {
	buffer := &bytes.Buffer{}
	var invalidType int

	if err := Write(buffer, invalidType); err == nil {
		t.Fatalf("got no error, expected: %q", ErrInvalidType)
	}
}
