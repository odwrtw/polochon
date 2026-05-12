package index

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestUpsertSubtitles(t *testing.T) {
	s1fr := &polochon.Subtitle{File: polochon.File{Size: 1000}, Lang: polochon.FR}
	s2fr := &polochon.Subtitle{File: polochon.File{Size: 2000}, Lang: polochon.FR}
	s1en := &polochon.Subtitle{File: polochon.File{Size: 3000}, Lang: polochon.EN}

	tt := []struct {
		name     string
		subs     []*polochon.Subtitle
		sub      *polochon.Subtitle
		expected []*polochon.Subtitle
	}{
		{
			name:     "no sub",
			expected: nil,
		},
		{
			name:     "no new sub",
			subs:     []*polochon.Subtitle{s1fr, s1en},
			expected: []*polochon.Subtitle{s1fr, s1en},
		},
		{
			name:     "new lang",
			subs:     []*polochon.Subtitle{s1fr},
			sub:      s1en,
			expected: []*polochon.Subtitle{s1fr, s1en},
		},
		{
			name:     "replace lang",
			sub:      s2fr,
			subs:     []*polochon.Subtitle{s1fr, s1en},
			expected: []*polochon.Subtitle{s2fr, s1en},
		},
		{
			name:     "empty subs",
			sub:      s1fr,
			expected: []*polochon.Subtitle{s1fr},
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
