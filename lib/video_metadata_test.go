package polochon

import (
	"reflect"
	"testing"
	"time"
)

func TestVideoMetatdataUpdate(t *testing.T) {
	vm := VideoMetadata{
		DateAdded:         time.Now(),
		Quality:           Quality1080p,
		ReleaseGroup:      "YTS",
		AudioCodec:        "AAC",
		VideoCodec:        "h264",
		Container:         "mkv",
		EmbeddedSubtitles: []Language{FR, EN},
	}

	newTime := time.Now().Add(1 * time.Second)

	tt := []struct {
		name     string
		metadata *VideoMetadata
		expected VideoMetadata
	}{
		{
			name:     "no metadata",
			expected: vm,
		},
		{
			name:     "no change",
			expected: vm,
			metadata: &vm,
		},
		{
			name: "everything new",
			expected: VideoMetadata{
				DateAdded:         newTime,
				Quality:           Quality3D,
				ReleaseGroup:      "YGG",
				AudioCodec:        "AAC-HD",
				VideoCodec:        "mp4",
				Container:         "new",
				EmbeddedSubtitles: []Language{FR},
			},
			metadata: &VideoMetadata{
				DateAdded:         newTime,
				Quality:           Quality3D,
				ReleaseGroup:      "YGG",
				AudioCodec:        "AAC-HD",
				VideoCodec:        "mp4",
				Container:         "new",
				EmbeddedSubtitles: []Language{FR},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := vm
			got.Update(tc.metadata)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("expected %+v, got %+v", tc.expected, got)
			}
		})
	}
}
