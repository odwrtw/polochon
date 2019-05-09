package nfo

import (
	"bytes"
	"testing"
	"time"
)

func init() {
	now = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2019-05-07T12:00:00Z")
		return t
	}
}

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
