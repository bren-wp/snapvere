package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"snapvera/internal/avi"
	w "snapvera/internal/win"
)

type recordingSession struct {
	stop chan struct{}
	done chan struct{}
	once sync.Once
	path string
	rect Rect
}

var (
	recordingFlag atomic.Bool
	recordingMu   sync.Mutex
	recorder      *recordingSession
)

type recordSurface struct {
	cap    *Capture
	screen uintptr
	mem    uintptr
	old    uintptr
}

func videoDir() string {
	var b [520]uint16
	r, _, _ := w.ProcSHGetFolderPathW.Call(0, w.CSIDL_MYVIDEO|w.CSIDL_FLAG_CREATE, 0, w.SHGFP_TYPE_CURRENT, uintptr(unsafe.Pointer(&b[0])))
	base := ""
	if int32(r) >= 0 {
		base = syscall.UTF16ToString(b[:])
	}
	return ensureMediaDir(base, "Videos")
}

func newRecordSurface(r Rect) (*recordSurface, error) {
	if !validRect(r) {
		return nil, fmt.Errorf("invalid recording size %dx%d", r.W, r.H)
	}
	sdc, _, e := w.ProcGetDC.Call(0)
	if sdc == 0 {
		return nil, fmt.Errorf("GetDC: %w", e)
	}
	mdc, _, e := w.ProcCreateCompatibleDC.Call(sdc)
	if mdc == 0 {
		w.ProcReleaseDC.Call(0, sdc)
		return nil, fmt.Errorf("CreateCompatibleDC: %w", e)
	}
	bmi := w.BITMAPINFO{}
	bmi.Header.Size = uint32(unsafe.Sizeof(w.BITMAPINFOHEADER{}))
	bmi.Header.Width = r.W
	bmi.Header.Height = -r.H
	bmi.Header.Planes = 1
	bmi.Header.BitCount = 32
	bmi.Header.Compression = w.BI_RGB
	bmi.Header.SizeImage = uint32(r.W * r.H * 4)
	var bits unsafe.Pointer
	hb, _, e := w.ProcCreateDIBSection.Call(sdc, uintptr(unsafe.Pointer(&bmi)), w.DIB_RGB_COLORS, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hb == 0 || bits == nil {
		w.ProcDeleteDC.Call(mdc)
		w.ProcReleaseDC.Call(0, sdc)
		return nil, fmt.Errorf("CreateDIBSection: %w", e)
	}
	old, _, _ := w.ProcSelectObject.Call(mdc, hb)
	c := &Capture{H: w.HBITMAP(hb), Bits: bits, W: r.W, Hh: r.H, Stride: r.W * 4, BMI: bmi}
	return &recordSurface{cap: c, screen: sdc, mem: mdc, old: old}, nil
}

func (s *recordSurface) close() {
	if s == nil {
		return
	}
	if s.mem != 0 {
		w.ProcSelectObject.Call(s.mem, s.old)
	}
	if s.cap != nil {
		s.cap.Close()
	}
	if s.mem != 0 {
		w.ProcDeleteDC.Call(s.mem)
	}
	if s.screen != 0 {
		w.ProcReleaseDC.Call(0, s.screen)
	}
}

func (s *recordSurface) grab(r Rect) error {
	ok, _, e := w.ProcBitBlt.Call(s.mem, 0, 0, uintptr(r.W), uintptr(r.H), s.screen, uintptr(r.X), uintptr(r.Y), w.SRCCOPY|w.CAPTUREBLT)
	if ok == 0 {
		ok, _, e = w.ProcBitBlt.Call(s.mem, 0, 0, uintptr(r.W), uintptr(r.H), s.screen, uintptr(r.X), uintptr(r.Y), w.SRCCOPY)
	}
	if ok == 0 {
		return fmt.Errorf("BitBlt recording: %w", e)
	}
	return nil
}

func jpegFrame(c *Capture, quality int, buf *bytes.Buffer) error {
	pix := unsafe.Slice((*byte)(c.Bits), int(c.Stride*c.Hh))
	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+2] = pix[i+2], pix[i]
		pix[i+3] = 255
	}
	img := &image.RGBA{Pix: pix, Stride: int(c.Stride), Rect: image.Rect(0, 0, int(c.W), int(c.Hh))}
	buf.Reset()
	err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality})
	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+2] = pix[i+2], pix[i]
	}
	return err
}

func recordingActive() bool { return recordingFlag.Load() }

func currentRecording() *recordingSession {
	recordingMu.Lock()
	defer recordingMu.Unlock()
	return recorder
}

func startRecording(mode string) {
	if recordingActive() {
		notify("Snapvera", t("recording_already", "Recording is already active."), w.NIIF_INFO)
		return
	}
	if captureBusy {
		notify("Snapvera", t("capture_busy", "A capture is already active."), w.NIIF_WARNING)
		return
	}

	var r Rect
	if mode == "region" {
		rr, ok, err := selectRegion()
		if err != nil {
			showError(err.Error(), errorCode(err))
			return
		}
		if !ok {
			return
		}
		r = rr
	} else {
		r = virtualDesktop()
		if !validRect(r) {
			showError("invalid recording area", 0)
			return
		}
	}

	// Only one recording may own the capture resources. CompareAndSwap closes
	// the small race between checking the flag and launching the goroutine.
	if !recordingFlag.CompareAndSwap(false, true) {
		notify("Snapvera", t("recording_already", "Recording is already active."), w.NIIF_INFO)
		return
	}

	s := &recordingSession{
		stop: make(chan struct{}),
		done: make(chan struct{}),
		path: defaultVideoFile(mode),
		rect: r,
	}
	recordingMu.Lock()
	recorder = s
	recordingMu.Unlock()
	refreshRecordingButtons()

	logf("recording start mode=%s rect=%+v path=%s", mode, r, s.path)
	notify("Snapvera", t("recording_started", "Screen recording started."), w.NIIF_INFO)
	go recordLoop(s)
}

func stopRecording() {
	s := currentRecording()
	if s == nil {
		return
	}
	s.once.Do(func() { close(s.stop) })
}

func stopRecordingAndWait(timeout time.Duration) {
	s := currentRecording()
	if s == nil {
		return
	}
	s.once.Do(func() { close(s.stop) })
	select {
	case <-s.done:
	case <-time.After(timeout):
		logf("recording shutdown timeout")
	}
}

func finishRecordingSession(s *recordingSession) {
	recordingMu.Lock()
	if recorder == s {
		recorder = nil
	}
	recordingMu.Unlock()
	recordingFlag.Store(false)
	// Marshal UI state refresh back to the tray-host thread instead of touching
	// settings controls directly from the recording goroutine.
	if mainHWND != 0 {
		w.ProcPostMessageW.Call(uintptr(mainHWND), wmRecordingState, 0, 0)
	}
	close(s.done)
}

func recordLoop(s *recordingSession) {
	// Registered first so it runs last, after writer/surface cleanup.
	defer finishRecordingSession(s)

	surface, err := newRecordSurface(s.rect)
	if err != nil {
		logf("recording init error: %v", err)
		notify("Snapvera", t("recording_failed", "Screen recording failed."), w.NIIF_ERROR)
		return
	}
	defer surface.close()

	fps, q := recordingParams()
	writer, err := avi.New(s.path, s.rect.W, s.rect.H, fps, q)
	if err != nil {
		logf("AVI create error: %v", err)
		notify("Snapvera", t("recording_failed", "Screen recording failed."), w.NIIF_ERROR)
		return
	}

	success := false
	defer func() {
		cerr := writer.Close()
		if cerr != nil {
			logf("AVI close error: %v", cerr)
			success = false
		}
		if !success {
			_ = os.Remove(s.path)
		} else {
			logf("recording saved path=%s", s.path)
			addHistoryEntry(s.path, "video", s.rect.W, s.rect.H)
			notify("Snapvera", t("recording_saved", "Recording saved.")+" "+filepath.Base(s.path), w.NIIF_INFO)
		}
		if int64(s.rect.W)*int64(s.rect.H) >= 8_000_000 {
			debug.FreeOSMemory()
		}
	}()

	frameDur := time.Second / time.Duration(fps)
	var buf bytes.Buffer
	frames := 0
	started := time.Now()
	for {
		frameStart := time.Now()
		select {
		case <-s.stop:
			success = frames > 0
			return
		default:
		}
		if time.Since(started) > 2*time.Hour {
			success = frames > 0
			return
		}
		if err = surface.grab(s.rect); err != nil {
			logf("recording grab error: %v", err)
			return
		}
		if err = jpegFrame(surface.cap, q, &buf); err != nil {
			logf("recording jpeg error: %v", err)
			return
		}
		if err = writer.AddJPEG(buf.Bytes()); err != nil {
			logf("recording write error: %v", err)
			return
		}
		frames++
		// Classic AVI uses 32-bit RIFF/index offsets. Stop safely before the
		// 4 GiB boundary instead of producing a corrupt recording.
		if writer.BytesWritten() >= 3_700_000_000 {
			logf("recording reached safe AVI size limit")
			success = true
			return
		}
		if elapsed := time.Since(frameStart); elapsed < frameDur {
			timer := time.NewTimer(frameDur - elapsed)
			select {
			case <-s.stop:
				if !timer.Stop() {
					<-timer.C
				}
				success = frames > 0
				return
			case <-timer.C:
			}
		}
	}
}

func recordingBufferSelfTest(c *Capture) error {
	if c == nil || c.Bits == nil || c.W < 2 || c.Hh < 2 || c.Stride < c.W*4 {
		return fmt.Errorf("invalid synthetic recording buffer")
	}
	p := filepath.Join(os.TempDir(), "Snapvera-recording-headless-self-test.avi")
	wr, err := avi.New(p, c.W, c.Hh, 5, 60)
	if err != nil {
		return err
	}
	closed := false
	defer func() {
		if !closed {
			_ = wr.Close()
		}
		_ = os.Remove(p)
	}()
	var b bytes.Buffer
	for i := 0; i < 2; i++ {
		if err = jpegFrame(c, 60, &b); err != nil {
			return err
		}
		if err = wr.AddJPEG(b.Bytes()); err != nil {
			return err
		}
	}
	if err = wr.Close(); err != nil {
		return err
	}
	closed = true
	st, err := os.Stat(p)
	if err != nil {
		return err
	}
	if st.Size() < 512 {
		return fmt.Errorf("headless recording self-test produced a small AVI")
	}
	return nil
}

func recordingSelfTest() error {
	v := virtualDesktop()
	r := Rect{X: v.X, Y: v.Y, W: min32(96, v.W), H: min32(64, v.H)}
	s, err := newRecordSurface(r)
	if err != nil {
		return err
	}
	defer s.close()
	p := filepath.Join(os.TempDir(), "Snapvera-recording-self-test.avi")
	wr, err := avi.New(p, r.W, r.H, 5, 60)
	if err != nil {
		return err
	}
	var b bytes.Buffer
	for i := 0; i < 2; i++ {
		if err = s.grab(r); err != nil {
			_ = wr.Close()
			_ = os.Remove(p)
			return err
		}
		if err = jpegFrame(s.cap, 60, &b); err != nil {
			_ = wr.Close()
			_ = os.Remove(p)
			return err
		}
		if err = wr.AddJPEG(b.Bytes()); err != nil {
			_ = wr.Close()
			_ = os.Remove(p)
			return err
		}
	}
	if err = wr.Close(); err != nil {
		_ = os.Remove(p)
		return err
	}
	st, err := os.Stat(p)
	_ = os.Remove(p)
	if err != nil {
		return err
	}
	if st.Size() < 512 {
		return fmt.Errorf("recording self-test produced a small AVI")
	}
	return nil
}
