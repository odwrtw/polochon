package polochon

import "testing"

func TestIsAllowedQuality(t *testing.T) {
	for _, allowedQuality := range []Quality{
		Quality480p,
		Quality720p,
		Quality1080p,
		Quality2160p,
		Quality3D,
	} {
		if !allowedQuality.IsAllowed() {
			t.Errorf("Quality should be allowed: %q", allowedQuality)
		}
	}

	for _, invalidQuality := range []Quality{
		Quality("yo"),
		Quality("mama"),
	} {
		if invalidQuality.IsAllowed() {
			t.Errorf("Quality should not be allowed: %q", invalidQuality)
		}
	}
}
