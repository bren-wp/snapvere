package avi

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"testing"
)

func TestWriter(t *testing.T) {
	p := t.TempDir() + "/test.avi"
	w, err := New(p, 64, 48, 10, 75)
	if err != nil {
		t.Fatal(err)
	}
	for f := 0; f < 5; f++ {
		img := image.NewRGBA(image.Rect(0, 0, 64, 48))
		for y := 0; y < 48; y++ {
			for x := 0; x < 64; x++ {
				img.Set(x, y, color.RGBA{uint8(x * 4), uint8(y * 5), uint8(f * 40), 255})
			}
		}
		var b bytes.Buffer
		if err := jpeg.Encode(&b, img, &jpeg.Options{Quality: 75}); err != nil {
			t.Fatal(err)
		}
		if err := w.AddJPEG(b.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(p)
	if err != nil || st.Size() < 1000 {
		t.Fatalf("bad AVI size: %v %v", st, err)
	}
}
