package polochon

import "testing"

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
