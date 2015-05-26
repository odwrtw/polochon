package polochon

import "testing"

func TestVideoType(t *testing.T) {
	for expected, s := range map[VideoType]string{
		MovieType:       "movie",
		ShowEpisodeType: "episode",
		ShowType:        "show",
	} {
		got, err := videoType(s)
		if err != nil {
			t.Errorf("Should get video type, instead we've got %q", err)
		}

		if got != expected {
			t.Errorf("Expected %q got %q", expected, got)
		}
	}

	_, err := videoType("invalidType")
	if err != ErrInvalidVideoType {
		t.Errorf("Expected %q got %q", ErrInvalidVideoType, err)
	}
}

func TestQualityFromString(t *testing.T) {
	for expected, s := range map[Quality]string{
		Quality480p:  "480p",
		Quality720p:  "720p",
		Quality1080p: "1080p",
		Quality3D:    "3D",
	} {
		got, err := GetQuality(s)
		if err != nil {
			t.Errorf("Should get quality, instead we've got %q", err)
		}

		if got != expected {
			t.Errorf("Expected %q got %q", expected, got)
		}
	}

	_, err := GetQuality("invalidType")
	if err != ErrInvalidQuality {
		t.Errorf("Expected %q got %q", ErrInvalidQuality, err)
	}
}
