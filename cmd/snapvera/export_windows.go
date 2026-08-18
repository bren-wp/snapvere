package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"

	"snapvera/internal/naming"
	w "snapvera/internal/win"
)

var exportCounter uint64
var invalidFilename = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]+`)

func exportExtension() string {
	if strings.HasPrefix(prefs.ExportPreset, "jpg") {
		return ".jpg"
	}
	return ".png"
}
func exportPresetLabel() string {
	switch prefs.ExportPreset {
	case "jpg-high":
		return "JPEG 92%"
	case "jpg-balanced":
		return "JPEG 82%"
	case "jpg-small":
		return "JPEG 68%"
	default:
		return "PNG"
	}
}
func namingPresetLabel() string {
	switch prefs.NamePreset {
	case "compact":
		return "IMG/VID + YYYYMMDD-HHMMSS.mmm"
	case "timestamp":
		return "Screenshot/Video + Unix time"
	case "technical":
		return "Type + mode + timestamp"
	default:
		return "Screenshot/Video + date-time"
	}
}
func recordingPresetLabel() string {
	switch prefs.RecordingPreset {
	case "compact":
		return "Compact · 8 FPS · Q60"
	case "smooth":
		return "Smooth · 15 FPS · Q82"
	default:
		return "Balanced · 12 FPS · Q74"
	}
}
func recordingParams() (fps, quality int) {
	switch prefs.RecordingPreset {
	case "compact":
		return 8, 60
	case "smooth":
		return 15, 82
	default:
		return 12, 74
	}
}
func sanitizeBase(s string) string {
	s = invalidFilename.ReplaceAllString(strings.TrimSpace(s), "-")
	s = strings.Trim(s, ". ")
	if len(s) > 150 {
		s = s[:150]
	}
	if s == "" {
		s = "Snapvera"
	}
	return s
}
func fileBase(kind, mode string) string {
	n := atomic.AddUint64(&exportCounter, 1)
	mode = sanitizeBase(strings.ToLower(mode))
	return sanitizeBase(naming.Base(kind, mode, prefs.NamePreset, time.Now(), n))
}
func uniqueOutputPath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}
	dir := filepath.Dir(path)
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(filepath.Base(path), ext)
	for i := 2; i <= 9999; i++ {
		candidate := filepath.Join(dir, base+"-"+strconv.Itoa(i)+ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return filepath.Join(dir, base+"-"+strconv.FormatInt(time.Now().UnixNano(), 10)+ext)
}

func defaultImageFile(mode string) string {
	return uniqueOutputPath(filepath.Join(pictureDir(), fileBase("image", mode)+exportExtension()))
}
func defaultVideoFile(mode string) string {
	return uniqueOutputPath(filepath.Join(videoDir(), fileBase("video", mode)+".avi"))
}

func saveImageAtomic(c *Capture, path string, encode func(io.Writer, *image.RGBA) error) error {
	if c == nil || c.Bits == nil || !validRect(Rect{W: c.W, H: c.Hh}) || c.Stride < c.W*4 {
		return fmt.Errorf("invalid image buffer")
	}
	dataSize := int64(c.Stride) * int64(c.Hh)
	if dataSize <= 0 || dataSize > maxPixels*4 {
		return fmt.Errorf("invalid image buffer size: %d", dataSize)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".snapvera-image-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpName)
		}
	}()

	pix := unsafe.Slice((*byte)(c.Bits), int(dataSize))
	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+2] = pix[i+2], pix[i]
		// Screen-capture DIB alpha is not meaningful. Normalize it so PNG/JPEG
		// encoders never interpret captured pixels as transparent.
		pix[i+3] = 255
	}
	defer func() {
		for i := 0; i < len(pix); i += 4 {
			pix[i], pix[i+2] = pix[i+2], pix[i]
		}
	}()
	img := &image.RGBA{Pix: pix, Stride: int(c.Stride), Rect: image.Rect(0, 0, int(c.W), int(c.Hh))}
	if err := encode(tmp, img); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	r, _, moveErr := w.ProcMoveFileExW.Call(
		uintptr(unsafe.Pointer(w.UTF16(tmpName))),
		uintptr(unsafe.Pointer(w.UTF16(path))),
		w.MOVEFILE_REPLACE_EXISTING|w.MOVEFILE_WRITE_THROUGH,
	)
	if r == 0 {
		return fmt.Errorf("replace image file: %w", moveErr)
	}
	committed = true
	logf("save path=%s", path)
	return nil
}

func savePNG(c *Capture, path string) error {
	return saveImageAtomic(c, path, func(dst io.Writer, img *image.RGBA) error {
		return png.Encode(dst, img)
	})
}

func saveJPEG(c *Capture, path string, quality int) error {
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}
	return saveImageAtomic(c, path, func(dst io.Writer, img *image.RGBA) error {
		return jpeg.Encode(dst, img, &jpeg.Options{Quality: quality})
	})
}
func saveByPreset(c *Capture, path string) error {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".jpg" || ext == ".jpeg" {
		q := 82
		switch prefs.ExportPreset {
		case "jpg-high":
			q = 92
		case "jpg-small":
			q = 68
		}
		return saveJPEG(c, path, q)
	}
	return savePNG(c, path)
}
