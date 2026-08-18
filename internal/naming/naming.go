package naming

import (
	"strconv"
	"strings"
	"time"
)

// Base returns the user-visible base filename without an extension.
// Image and video names are intentionally distinct in every preset.
func Base(kind, mode, preset string, now time.Time, seq uint64) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "capture"
	}
	label := "Screenshot"
	short := "IMG"
	if kind == "video" {
		label = "Video"
		short = "VID"
	}
	seqText := strconv.FormatUint(seq, 10)
	switch preset {
	case "compact":
		return "SV-" + short + "-" + now.Format("20060102-150405.000") + "-" + seqText
	case "timestamp":
		return "Snapvera-" + label + "-" + strconv.FormatInt(now.UnixMilli(), 10) + "-" + seqText
	case "technical":
		return "Snapvera-" + label + "-" + mode + "-" + now.Format("20060102-150405.000") + "-" + seqText
	default:
		return "Snapvera-" + label + "-" + now.Format("2006-01-02_15-04-05.000") + "-" + seqText
	}
}
