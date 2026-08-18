package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
	"unsafe"

	w "snapvera/internal/win"
)

type Rect struct{ X, Y, W, H int32 }

func virtualDesktop() Rect {
	x, _, _ := w.ProcGetSystemMetrics.Call(w.SM_XVIRTUALSCREEN)
	y, _, _ := w.ProcGetSystemMetrics.Call(w.SM_YVIRTUALSCREEN)
	ww, _, _ := w.ProcGetSystemMetrics.Call(w.SM_CXVIRTUALSCREEN)
	hh, _, _ := w.ProcGetSystemMetrics.Call(w.SM_CYVIRTUALSCREEN)
	return Rect{int32(x), int32(y), int32(ww), int32(hh)}
}
func validRect(r Rect) bool { return r.W > 1 && r.H > 1 && int64(r.W)*int64(r.H) <= maxPixels }

type Capture struct {
	H             w.HBITMAP
	Bits          unsafe.Pointer
	W, Hh, Stride int32
	BMI           w.BITMAPINFO
	Mode          string
}

func (c *Capture) Close() {
	if c != nil && c.H != 0 {
		w.ProcDeleteObject.Call(uintptr(c.H))
		c.H = 0
		c.Bits = nil
	}
}
func captureRect(r Rect) (*Capture, error) {
	if !validRect(r) {
		return nil, fmt.Errorf("invalid capture size %dx%d", r.W, r.H)
	}
	sdc, _, e := w.ProcGetDC.Call(0)
	if sdc == 0 {
		return nil, fmt.Errorf("GetDC: %w", e)
	}
	defer w.ProcReleaseDC.Call(0, sdc)
	mdc, _, e := w.ProcCreateCompatibleDC.Call(sdc)
	if mdc == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC: %w", e)
	}
	defer w.ProcDeleteDC.Call(mdc)
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
		return nil, fmt.Errorf("CreateDIBSection: %w", e)
	}
	old, _, _ := w.ProcSelectObject.Call(mdc, hb)
	defer w.ProcSelectObject.Call(mdc, old)
	ok, _, e := w.ProcBitBlt.Call(mdc, 0, 0, uintptr(r.W), uintptr(r.H), sdc, uintptr(r.X), uintptr(r.Y), w.SRCCOPY|w.CAPTUREBLT)
	if ok == 0 {
		logf("BitBlt CAPTUREBLT failed: %v; retrying SRCCOPY", e)
		ok, _, e = w.ProcBitBlt.Call(mdc, 0, 0, uintptr(r.W), uintptr(r.H), sdc, uintptr(r.X), uintptr(r.Y), w.SRCCOPY)
	}
	if ok == 0 {
		w.ProcDeleteObject.Call(hb)
		return nil, fmt.Errorf("BitBlt: %w", e)
	}
	return &Capture{H: w.HBITMAP(hb), Bits: bits, W: r.W, Hh: r.H, Stride: r.W * 4, BMI: bmi}, nil
}

var sel struct {
	start, end             w.POINT
	dragging, done, cancel bool
	hwnd                   w.HWND
	virtual                Rect
}
var overlayProcCB uintptr

func overlayProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case w.WM_LBUTTONDOWN:
		sel.dragging = true
		sel.start = w.POINT{X: w.XFromLP(lp), Y: w.YFromLP(lp)}
		sel.end = sel.start
		w.ProcSetCapture.Call(hwnd)
		w.ProcInvalidateRect.Call(hwnd, 0, 1)
		return 0
	case w.WM_MOUSEMOVE:
		if sel.dragging {
			sel.end = w.POINT{X: w.XFromLP(lp), Y: w.YFromLP(lp)}
			w.ProcInvalidateRect.Call(hwnd, 0, 1)
		}
		return 0
	case w.WM_LBUTTONUP:
		if sel.dragging {
			sel.end = w.POINT{X: w.XFromLP(lp), Y: w.YFromLP(lp)}
			sel.dragging = false
			w.ProcReleaseCapture.Call()
			sel.done = true
			w.ProcDestroyWindow.Call(hwnd)
		}
		return 0
	case w.WM_KEYDOWN:
		if wp == w.VK_ESCAPE {
			sel.cancel = true
			sel.done = true
			w.ProcDestroyWindow.Call(hwnd)
			return 0
		}
	case w.WM_ERASEBKGND:
		return 1
	case w.WM_PAINT:
		var ps w.PAINTSTRUCT
		dc, _, _ := w.ProcBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		var cr w.RECT
		w.ProcGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
		br, _, _ := w.ProcCreateSolidBrush.Call(w.RGB(10, 14, 22))
		w.ProcFillRect.Call(dc, uintptr(unsafe.Pointer(&cr)), br)
		w.ProcDeleteObject.Call(br)
		if sel.dragging || sel.start != sel.end {
			l := sel.start.X
			if sel.end.X < l {
				l = sel.end.X
			}
			t0 := sel.start.Y
			if sel.end.Y < t0 {
				t0 = sel.end.Y
			}
			r := sel.end.X
			if sel.start.X > r {
				r = sel.start.X
			}
			b := sel.end.Y
			if sel.start.Y > b {
				b = sel.start.Y
			}
			pen, _, _ := w.ProcCreatePen.Call(w.PS_SOLID, 3, w.RGB(47, 139, 253))
			hollow, _, _ := w.ProcGetStockObject.Call(w.HOLLOW_BRUSH)
			op, _, _ := w.ProcSelectObject.Call(dc, pen)
			ob, _, _ := w.ProcSelectObject.Call(dc, hollow)
			w.ProcRectangle.Call(dc, uintptr(l), uintptr(t0), uintptr(r), uintptr(b))
			w.ProcSelectObject.Call(dc, op)
			w.ProcSelectObject.Call(dc, ob)
			w.ProcDeleteObject.Call(pen)
		}
		w.ProcEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	}
	r, _, _ := w.ProcDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}
func selectRegion() (Rect, bool, error) {
	v := virtualDesktop()
	if !validRect(v) {
		logf("selectRegion invalid virtual=%+v", v)
		return Rect{}, false, fmt.Errorf("invalid virtual desktop %dx%d", v.W, v.H)
	}
	inst, _, _ := w.ProcGetModuleHandleW.Call(0)
	cls := w.UTF16("Snapvera.CaptureOverlay.Go.v1")
	wc := w.WNDCLASSEX{CbSize: uint32(unsafe.Sizeof(w.WNDCLASSEX{})), Style: w.CS_HREDRAW | w.CS_VREDRAW, LpfnWndProc: overlayProcCB, HInstance: w.HINSTANCE(inst), LpszClassName: cls}
	cur, _, _ := w.ProcLoadCursorW.Call(0, w.IntResource(w.IDC_CROSS))
	wc.HCursor = w.HCURSOR(cur)
	w.ProcRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, e := w.ProcCreateWindowExW.Call(w.WS_EX_TOPMOST|w.WS_EX_TOOLWINDOW|w.WS_EX_LAYERED, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(w.UTF16("Snapvera"))), w.WS_POPUP, uintptr(v.X), uintptr(v.Y), uintptr(v.W), uintptr(v.H), 0, 0, inst, 0)
	if hwnd == 0 {
		logf("overlay CreateWindowEx failed: %v", e)
		return Rect{}, false, fmt.Errorf("CreateWindowEx overlay: %w", e)
	}
	sel = struct {
		start, end             w.POINT
		dragging, done, cancel bool
		hwnd                   w.HWND
		virtual                Rect
	}{hwnd: w.HWND(hwnd), virtual: v}
	w.ProcSetLayeredWindowAttributes.Call(hwnd, 0, 100, w.LWA_ALPHA)
	applyWindowStyle(hwnd)
	w.ProcShowWindow.Call(hwnd, w.SW_SHOW)
	w.ProcUpdateWindow.Call(hwnd)
	w.ProcSetForegroundWindow.Call(hwnd)
	w.ProcSetFocus.Call(hwnd)
	logf("overlay shown virtual=%+v", v)
	var m w.MSG
	for !sel.done {
		r, _, _ := w.ProcGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		w.ProcTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		w.ProcDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	if sel.cancel {
		return Rect{}, false, nil
	}
	sx, ex := sel.start.X, sel.end.X
	if ex < sx {
		sx, ex = ex, sx
	}
	sy, ey := sel.start.Y, sel.end.Y
	if ey < sy {
		sy, ey = ey, sy
	}
	out := Rect{v.X + sx, v.Y + sy, ex - sx, ey - sy}
	logf("selection=%+v", out)
	if !validRect(out) {
		return Rect{}, false, fmt.Errorf("invalid selection %dx%d", out.W, out.H)
	}
	return out, true, nil
}

func captureActiveWindow() (Rect, error) {
	time.Sleep(120 * time.Millisecond)
	fh, _, _ := w.ProcGetForegroundWindow.Call()
	if fh == 0 || w.HWND(fh) == mainHWND || w.HWND(fh) == settingsHWND {
		fh = uintptr(lastExternal)
	}
	if fh == 0 {
		return Rect{}, fmt.Errorf("no foreground window")
	}
	var rr w.RECT
	gotRect := false
	if err := w.ProcDwmGetWindowAttribute.Find(); err == nil {
		hr, _, _ := w.ProcDwmGetWindowAttribute.Call(
			fh,
			w.DWMWA_EXTENDED_FRAME_BOUNDS,
			uintptr(unsafe.Pointer(&rr)),
			unsafe.Sizeof(rr),
		)
		gotRect = int32(hr) >= 0 && rr.Right > rr.Left && rr.Bottom > rr.Top
	}
	if !gotRect {
		ok, _, e := w.ProcGetWindowRect.Call(fh, uintptr(unsafe.Pointer(&rr)))
		if ok == 0 {
			return Rect{}, fmt.Errorf("GetWindowRect: %w", e)
		}
	}
	r := Rect{rr.Left, rr.Top, rr.Right - rr.Left, rr.Bottom - rr.Top}
	v := virtualDesktop()
	l := max32(r.X, v.X)
	t0 := max32(r.Y, v.Y)
	rrgt := min32(r.X+r.W, v.X+v.W)
	bot := min32(r.Y+r.H, v.Y+v.H)
	r = Rect{l, t0, rrgt - l, bot - t0}
	if !validRect(r) {
		return Rect{}, fmt.Errorf("invalid active-window rectangle %dx%d", r.W, r.H)
	}
	return r, nil
}
func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func doCapture(mode string, delay time.Duration) {
	if recordingActive() {
		notify("Snapvera", t("recording_already", "Recording is already active."), w.NIIF_INFO)
		return
	}
	if captureBusy {
		return
	}
	captureBusy = true
	defer func() { captureBusy = false }()
	nextMode, nextDelay := mode, delay
	for {
		logf("capture request mode=%s delay=%s", nextMode, nextDelay)
		if settingsHWND != 0 {
			w.ProcShowWindow.Call(uintptr(settingsHWND), w.SW_HIDE)
		}
		if nextDelay > 0 {
			time.Sleep(nextDelay)
		} else {
			time.Sleep(150 * time.Millisecond)
		}
		var r Rect
		switch nextMode {
		case "region":
			selectedRect, selected, err := selectRegion()
			if err != nil {
				showError(err.Error(), errorCode(err))
				return
			}
			if !selected {
				logf("region capture cancelled by user")
				return
			}
			r = selectedRect
		case "full":
			r = virtualDesktop()
			if !validRect(r) {
				err := fmt.Errorf("invalid virtual desktop %dx%d", r.W, r.H)
				showError(err.Error(), 0)
				return
			}
		case "window":
			windowRect, err := captureActiveWindow()
			if err != nil {
				showError(err.Error(), errorCode(err))
				return
			}
			r = windowRect
		default:
			return
		}
		cap, err := captureRect(r)
		if err != nil {
			showError(err.Error(), errorCode(err))
			return
		}
		cap.Mode = nextMode
		logf("capture success mode=%s rect=%+v size=%dx%d", nextMode, r, cap.W, cap.Hh)
		pixels := int64(cap.W) * int64(cap.Hh)
		retake := runEditor(cap)
		cap.Close()
		// Avoid a forced full GC after every small screenshot. Large captures are
		// explicitly returned to the OS because their DIB/editor buffers can be sizable.
		if pixels >= 8_000_000 {
			runtime.GC()
			debug.FreeOSMemory()
		}
		if retake {
			nextMode = "region"
			nextDelay = 0
			continue
		}
		return
	}
}

func addButton(parent uintptr, id int, text string, x, y, wid int32) uintptr {
	return addModernButton(parent, id, text, x, y, wid, modernButtonHeight)
}

func ensureMediaDir(base, fallbackName string) string {
	candidates := make([]string, 0, 3)
	if strings.TrimSpace(base) != "" {
		candidates = append(candidates, filepath.Join(base, "Snapvera"))
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		candidates = append(candidates, filepath.Join(home, fallbackName, "Snapvera"))
	}
	candidates = append(candidates, filepath.Join(appDataDir(), fallbackName))
	for _, d := range candidates {
		if err := os.MkdirAll(d, 0700); err == nil {
			return d
		} else {
			logf("media directory unavailable path=%s err=%v", d, err)
		}
	}
	d := filepath.Join(os.TempDir(), "Snapvera", fallbackName)
	_ = os.MkdirAll(d, 0700)
	return d
}

func pictureDir() string {
	var b [520]uint16
	r, _, _ := w.ProcSHGetFolderPathW.Call(0, w.CSIDL_MYPICTURES|w.CSIDL_FLAG_CREATE, 0, w.SHGFP_TYPE_CURRENT, uintptr(unsafe.Pointer(&b[0])))
	base := ""
	if int32(r) >= 0 {
		base = syscall.UTF16ToString(b[:])
	}
	return ensureMediaDir(base, "Pictures")
}
func defaultFile(c *Capture) string {
	mode := "capture"
	if c != nil && c.Mode != "" {
		mode = c.Mode
	}
	return defaultImageFile(mode)
}
func saveDefault(c *Capture) (string, error) {
	p := defaultFile(c)
	err := saveByPreset(c, p)
	if err == nil && c != nil {
		addHistoryEntry(p, "image", c.W, c.Hh)
	}
	return p, err
}
func multiSZ(parts ...string) []uint16 {
	out := make([]uint16, 0, 128)
	for _, part := range parts {
		u := syscall.StringToUTF16(part)
		out = append(out, u...)
	}
	out = append(out, 0)
	return out
}
func saveAs(c *Capture, owner uintptr) (string, bool) {
	p := defaultFile(c)
	buf := make([]uint16, 1024)
	copy(buf, syscall.StringToUTF16(p))
	filter := multiSZ("PNG image (*.png)", "*.png", "JPEG image (*.jpg)", "*.jpg;*.jpeg", "All files (*.*)", "*.*")
	title := syscall.StringToUTF16(t("save_as", "Save as"))
	defExt := "png"
	filterIndex := uint32(1)
	if strings.HasPrefix(prefs.ExportPreset, "jpg") {
		defExt = "jpg"
		filterIndex = 2
	}
	def := syscall.StringToUTF16(defExt)
	ofn := w.OPENFILENAME{StructSize: uint32(unsafe.Sizeof(w.OPENFILENAME{})), Owner: w.HWND(owner), Filter: &filter[0], FilterIndex: filterIndex, File: &buf[0], MaxFile: uint32(len(buf)), Title: &title[0], Flags: w.OFN_OVERWRITEPROMPT | w.OFN_NOCHANGEDIR | w.OFN_PATHMUSTEXIST, DefExt: &def[0]}
	r, _, _ := w.ProcGetSaveFileNameW.Call(uintptr(unsafe.Pointer(&ofn)))
	if r == 0 {
		return "", false
	}
	out := syscall.UTF16ToString(buf)
	if filepath.Ext(out) == "" {
		if ofn.FilterIndex == 2 {
			out += ".jpg"
		} else {
			out += ".png"
		}
	}
	if err := saveByPreset(c, out); err != nil {
		message(t("save_failed", "Save failed")+"\n"+err.Error(), w.MB_ICONERROR)
		return "", false
	}
	if c != nil {
		addHistoryEntry(out, "image", c.W, c.Hh)
	}
	return out, true
}
func copyClipboard(c *Capture, owner uintptr) error {
	headerSize := unsafe.Sizeof(w.BITMAPINFOHEADER{})
	dataSize := uintptr(c.Stride * c.Hh)
	total := headerSize + dataSize
	hg, _, e := w.ProcGlobalAlloc.Call(w.GMEM_MOVEABLE|w.GMEM_ZEROINIT, total)
	if hg == 0 {
		return fmt.Errorf("GlobalAlloc: %v", e)
	}
	ptr, _, e := w.ProcGlobalLock.Call(hg)
	if ptr == 0 {
		w.ProcGlobalFree.Call(hg)
		return fmt.Errorf("GlobalLock: %v", e)
	}
	basePtr := unsafe.Pointer(ptr)
	hdr := (*w.BITMAPINFOHEADER)(basePtr)
	*hdr = c.BMI.Header
	hdr.Height = c.Hh
	hdr.SizeImage = uint32(dataSize)
	dst := unsafe.Slice((*byte)(unsafe.Add(basePtr, headerSize)), int(dataSize))
	src := unsafe.Slice((*byte)(c.Bits), int(dataSize))
	stride := int(c.Stride)
	for y := 0; y < int(c.Hh); y++ {
		copy(dst[y*stride:(y+1)*stride], src[(int(c.Hh)-1-y)*stride:(int(c.Hh)-y)*stride])
	}
	w.ProcGlobalUnlock.Call(hg)
	ok, _, e := w.ProcOpenClipboard.Call(owner)
	if ok == 0 {
		w.ProcGlobalFree.Call(hg)
		return fmt.Errorf("OpenClipboard: %v", e)
	}
	defer w.ProcCloseClipboard.Call()
	w.ProcEmptyClipboard.Call()
	r, _, e := w.ProcSetClipboardData.Call(w.CF_DIB, hg)
	if r == 0 {
		w.ProcGlobalFree.Call(hg)
		return fmt.Errorf("SetClipboardData: %v", e)
	}
	logf("clipboard success %dx%d", c.W, c.Hh)
	return nil
}

var hostProcCB uintptr
var settingsProcCB uintptr
