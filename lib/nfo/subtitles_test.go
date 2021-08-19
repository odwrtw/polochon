package nfo

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestSubToLangs(t *testing.T) {
	subs := []*polochon.Subtitle{
		{
			Lang: polochon.FR,
		},
		{Lang: polochon.EN, Embedded: true},
	}

	expected := []polochon.Language{polochon.FR}
	got := subToLang(subs)
	if reflect.DeepEqual(got, expected) {
		t.Fatalf("failed to extract langs from subtitles")
	}
}
