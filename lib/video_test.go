package polochon

import "testing"

func TestIsAllowedQuality(t *testing.T) {
	for _, allowedQuality := range []Quality{
		Quality480p,
		Quality720p,
		Quality1080p,
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

func TestSlug(t *testing.T) {
	for expected, s := range map[string]string{
		"abcd-lol-pwet-123":  "abcd (lol) pwet! [123]",
		"abcd-lol-pwet-1234": "abcd-lol-pwet-1234",
		"20-wt-o":            "é %20 w@t \\o/",
	} {
		got := slug(s)

		if got != expected {
			t.Errorf("Expected %q got %q", expected, got)
		}
	}
}
