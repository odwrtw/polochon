package polochon

import (
	"testing"
)

func TestShortForm(t *testing.T) {
	for _, l := range []struct {
		expected string
		lang     Language
	}{
		{
			expected: "en",
			lang:     EN,
		},
		{
			expected: "fr",
			lang:     FR,
		},
		{
			expected: "pwet",
			lang:     Language("pwet"),
		},
	} {
		short := l.lang.ShortForm()
		if short != l.expected {
			t.Errorf("Expected %#v, got %#v", l.expected, short)
		}
	}
}
