package naming

import (
	"strings"
	"testing"
	"time"
)

func TestImageAndVideoAreAlwaysDistinct(t *testing.T) {
	now := time.Date(2026, 8, 15, 22, 30, 45, 123000000, time.UTC)
	for _, preset := range []string{"standard", "compact", "timestamp", "technical"} {
		img := Base("image", "region", preset, now, 7)
		vid := Base("video", "region", preset, now, 7)
		if img == vid {
			t.Fatalf("preset %q generated identical image/video names: %q", preset, img)
		}
		if !strings.Contains(img, "IMG") && !strings.Contains(img, "Screenshot") {
			t.Fatalf("image name does not identify screenshot type: %q", img)
		}
		if !strings.Contains(vid, "VID") && !strings.Contains(vid, "Video") {
			t.Fatalf("video name does not identify video type: %q", vid)
		}
	}
}

func TestTechnicalIncludesCaptureMode(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	got := Base("video", "full", "technical", now, 3)
	if !strings.Contains(got, "-Video-full-") {
		t.Fatalf("technical video name missing mode: %q", got)
	}
}
