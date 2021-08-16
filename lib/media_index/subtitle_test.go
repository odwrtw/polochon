package index

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestUpsertSubtitles(t *testing.T) {
	s1fr := &Subtitle{Lang: polochon.FR, Size: 1000}
	s2fr := &Subtitle{Lang: polochon.FR, Size: 2000}
	s1en := &Subtitle{Lang: polochon.EN, Size: 3000}

	tt := []struct {
		name     string
		subs     []*Subtitle
		sub      *Subtitle
		expected []*Subtitle
	}{
		{
			name:     "no sub",
			expected: nil,
		},
		{
			name:     "no new sub",
			subs:     []*Subtitle{s1fr, s1en},
			expected: []*Subtitle{s1fr, s1en},
		},
		{
			name:     "new lang",
			subs:     []*Subtitle{s1fr},
			sub:      s1en,
			expected: []*Subtitle{s1fr, s1en},
		},
		{
			name:     "replace lang",
			sub:      s2fr,
			subs:     []*Subtitle{s1fr, s1en},
			expected: []*Subtitle{s2fr, s1en},
		},
		{
			name:     "empty subs",
			sub:      s1fr,
			expected: []*Subtitle{s1fr},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := upsertSubtitle(tc.subs, tc.sub)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("expected %#v, got %#v", tc.expected, got)
			}
		})
	}
}
