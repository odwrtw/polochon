package polochon

import "testing"

func TestStringToQuality(t *testing.T) {
	for _, quality := range []Quality{Quality480p, Quality720p, Quality1080p, Quality2160p, Quality3D} {
		t.Run(string(quality), func(t *testing.T) {
			got, err := StringToQuality(string(quality))
			if err != nil || got == nil || *got != quality || !got.IsAllowed() {
				t.Fatalf("StringToQuality = %v, %v; want %s", got, err, quality)
			}
		})
	}
	for _, value := range []string{"", "unknown", "1440p", "4K"} {
		if got, err := StringToQuality(value); err == nil || got != nil {
			t.Fatalf("StringToQuality(%q) = %v, %v; want error", value, got, err)
		}
	}
}

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
