package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"snapvera/internal/buildinfo"
	"snapvera/internal/i18n"
	"snapvera/internal/installtx"
	"snapvera/internal/resources"
	w "snapvera/internal/win"
)

const (
	idInstall       = 2101
	idCancel        = 2102
	idStartupToggle = 2103
)

//go:embed payload-app.exe
var appBytes []byte

//go:embed payload-uninstall.exe
var uninstallBytes []byte

var iconBytes = resources.Icon
var tr map[string]string

func t(key, fallback string) string {
	if v := tr[key]; v != "" {
		return v
	}
	return fallback
}
func initI18N() {
	locale := ""
	var buf [85]uint16
	if r, _, _ := w.ProcGetUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), 85); r != 0 {
		locale = syscall.UTF16ToString(buf[:])
	}
	_, tr = i18n.Resolve(locale)
}

var procCB uintptr
var win w.HWND
var installBtn, cancelBtn, startupBtn uintptr
var statusText string
var installed bool
var appPath string
var fontRegular, fontTitle w.HFONT
var appIcon w.HICON
var startWithWindows = true

func loadIcon() w.HICON {
	if len(iconBytes) < 22 {
		return 0
	}
	count := int(iconBytes[4]) | int(iconBytes[5])<<8
	bestOff, bestSize, bestArea := 0, 0, -1
	for i := 0; i < count; i++ {
		o := 6 + i*16
		if o+16 > len(iconBytes) {
			break
		}
		ww := int(iconBytes[o])
		hh := int(iconBytes[o+1])
		if ww == 0 {
			ww = 256
		}
		if hh == 0 {
			hh = 256
		}
		sz := int(iconBytes[o+8]) | int(iconBytes[o+9])<<8 | int(iconBytes[o+10])<<16 | int(iconBytes[o+11])<<24
		off := int(iconBytes[o+12]) | int(iconBytes[o+13])<<8 | int(iconBytes[o+14])<<16 | int(iconBytes[o+15])<<24
		if off >= 0 && sz > 0 && off+sz <= len(iconBytes) && ww*hh > bestArea {
			bestOff, bestSize, bestArea = off, sz, ww*hh
		}
	}
	if bestSize == 0 {
		return 0
	}
	r, _, _ := w.ProcCreateIconFromResourceEx.Call(uintptr(unsafe.Pointer(&iconBytes[bestOff])), uintptr(bestSize), 1, 0x00030000, 0, 0, 0)
	return w.HICON(r)
}
func ensureFonts() {
	if fontRegular == 0 {
		r, _, _ := w.ProcCreateFontW.Call(^uintptr(15), 0, 0, 0, w.FW_NORMAL, 0, 0, 0, w.DEFAULT_CHARSET, w.OUT_DEFAULT_PRECIS, w.CLIP_DEFAULT_PRECIS, w.CLEARTYPE_QUALITY, w.DEFAULT_PITCH|w.FF_DONTCARE, uintptr(unsafe.Pointer(w.UTF16("Segoe UI"))))
		fontRegular = w.HFONT(r)
	}
	if fontTitle == 0 {
		r, _, _ := w.ProcCreateFontW.Call(^uintptr(24), 0, 0, 0, w.FW_SEMIBOLD, 0, 0, 0, w.DEFAULT_CHARSET, w.OUT_DEFAULT_PRECIS, w.CLIP_DEFAULT_PRECIS, w.CLEARTYPE_QUALITY, w.DEFAULT_PITCH|w.FF_DONTCARE, uintptr(unsafe.Pointer(w.UTF16("Segoe UI"))))
		fontTitle = w.HFONT(r)
	}
}
func button(parent uintptr, id int, text string, x, y, wid int32) uintptr {
	inst, _, _ := w.ProcGetModuleHandleW.Call(0)
	h, _, _ := w.ProcCreateWindowExW.Call(0, uintptr(unsafe.Pointer(w.UTF16("BUTTON"))), uintptr(unsafe.Pointer(w.UTF16(text))), w.WS_CHILD|w.WS_VISIBLE|w.WS_TABSTOP|w.BS_OWNERDRAW, uintptr(x), uintptr(y), uintptr(wid), 56, parent, uintptr(id), inst, 0)
	w.ProcSendMessageW.Call(h, w.WM_SETFONT, uintptr(fontRegular), 1)
	w.ProcSetWindowTheme.Call(h, uintptr(unsafe.Pointer(w.UTF16("Explorer"))), 0)
	return h
}
func regSetString(key uintptr, name, value string) bool {
	u := syscall.StringToUTF16(value)
	var np uintptr
	if name != "" {
		np = uintptr(unsafe.Pointer(w.UTF16(name)))
	}
	r, _, _ := w.ProcRegSetValueExW.Call(key, np, 0, w.REG_SZ, uintptr(unsafe.Pointer(&u[0])), uintptr(len(u)*2))
	return r == 0
}
func regSetDWORD(key uintptr, name string, value uint32) bool {
	r, _, _ := w.ProcRegSetValueExW.Call(key, uintptr(unsafe.Pointer(w.UTF16(name))), 0, w.REG_DWORD, uintptr(unsafe.Pointer(&value)), 4)
	return r == 0
}
func createKey(path string) (uintptr, bool) {
	var h uintptr
	var disp uint32
	r, _, _ := w.ProcRegCreateKeyExW.Call(w.HKEYCurrentUser(), uintptr(unsafe.Pointer(w.UTF16(path))), 0, 0, 0, w.KEY_SET_VALUE, 0, uintptr(unsafe.Pointer(&h)), uintptr(unsafe.Pointer(&disp)))
	return h, r == 0
}
func registerInstall(dir, app, uninst string) error {
	if h, ok := createKey(`Software\Microsoft\Windows\CurrentVersion\App Paths\Snapvera.exe`); ok {
		defer w.ProcRegCloseKey.Call(h)
		if !regSetString(h, "", app) || !regSetString(h, "Path", dir) {
			return fmt.Errorf("unable to register App Paths")
		}
	} else {
		return fmt.Errorf("unable to create App Paths registration")
	}
	if h, ok := createKey(`Software\Microsoft\Windows\CurrentVersion\Uninstall\Snapvera`); ok {
		defer w.ProcRegCloseKey.Call(h)
		estimatedKB := uint32((len(appBytes) + len(uninstallBytes) + 1023) / 1024)
		pairs := [][2]string{
			{"DisplayName", buildinfo.Name},
			{"DisplayVersion", buildinfo.Version},
			{"Publisher", buildinfo.Publisher},
			{"URLInfoAbout", buildinfo.Website},
			{"URLUpdateInfo", buildinfo.Website},
			{"InstallLocation", dir},
			{"DisplayIcon", app + ",0"},
			{"UninstallString", fmt.Sprintf("\"%s\"", uninst)},
			{"InstallDate", time.Now().Format("20060102")},
		}
		for _, pair := range pairs {
			if !regSetString(h, pair[0], pair[1]) {
				return fmt.Errorf("unable to register %s", pair[0])
			}
		}
		if !regSetDWORD(h, "EstimatedSize", estimatedKB) || !regSetDWORD(h, "NoModify", 1) || !regSetDWORD(h, "NoRepair", 1) {
			return fmt.Errorf("unable to register uninstall metadata")
		}
	} else {
		return fmt.Errorf("unable to create uninstall registration")
	}
	return nil
}

func configureStartup(app string, enabled bool) error {
	h, ok := createKey(`Software\Microsoft\Windows\CurrentVersion\Run`)
	if !ok {
		return fmt.Errorf("unable to open Windows startup registration")
	}
	defer w.ProcRegCloseKey.Call(h)
	if !enabled {
		r, _, _ := w.ProcRegDeleteValueW.Call(h, uintptr(unsafe.Pointer(w.UTF16("Snapvera"))))
		if r != 0 && r != w.ERROR_FILE_NOT_FOUND {
			return fmt.Errorf("unable to remove Windows startup entry")
		}
		return nil
	}
	if !regSetString(h, "Snapvera", fmt.Sprintf("\"%s\" --background", app)) {
		return fmt.Errorf("unable to register Windows startup entry")
	}
	return nil
}

func installPayloads(appPath, uninstallPath string) error {
	return installtx.ReplaceAll([]installtx.File{
		{Path: appPath, Data: appBytes, Mode: 0700},
		{Path: uninstallPath, Data: uninstallBytes, Mode: 0700},
	})
}

func waitForOldInstance() bool {
	for i := 0; i < 50; i++ {
		h, _, _ := w.ProcFindWindowW.Call(uintptr(unsafe.Pointer(w.UTF16("Snapvera.TrayHost.Go.v2"))), 0)
		if h == 0 {
			return true
		}
		if i == 0 {
			w.ProcPostMessageW.Call(h, w.WM_CLOSE, 0, 0)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
func doInstall() {
	w.ProcEnableWindow.Call(installBtn, 0)
	statusText = t("install", "Install") + "…"
	w.ProcInvalidateRect.Call(uintptr(win), 0, 1)
	base := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
	if base == "" {
		statusText = "Installation failed: LOCALAPPDATA is unavailable."
		w.ProcEnableWindow.Call(installBtn, 1)
		w.ProcInvalidateRect.Call(uintptr(win), 0, 1)
		return
	}
	dir := filepath.Join(base, "Programs", "Snapvera")
	if err := os.MkdirAll(dir, 0700); err != nil {
		statusText = "Installation failed: " + err.Error()
		w.ProcEnableWindow.Call(installBtn, 1)
		w.ProcInvalidateRect.Call(uintptr(win), 0, 1)
		return
	}
	if !waitForOldInstance() {
		statusText = "Installation failed: close the running Snapvera instance and try again."
		w.ProcEnableWindow.Call(installBtn, 1)
		w.ProcInvalidateRect.Call(uintptr(win), 0, 1)
		return
	}
	app := filepath.Join(dir, "Snapvera.exe")
	uninst := filepath.Join(dir, "Uninstall.exe")
	if err := installPayloads(app, uninst); err != nil {
		statusText = "Installation failed: " + err.Error()
		w.ProcEnableWindow.Call(installBtn, 1)
		w.ProcInvalidateRect.Call(uintptr(win), 0, 1)
		return
	}
	warnings := make([]string, 0, 2)
	if err := registerInstall(dir, app, uninst); err != nil {
		warnings = append(warnings, "Windows registration: "+err.Error())
	}
	if err := configureStartup(app, startWithWindows); err != nil {
		warnings = append(warnings, "Startup preference: "+err.Error())
	}
	appPath = app
	installed = true
	statusText = t("installed", "Installed") + " — Snapvera"
	if len(warnings) > 0 {
		statusText += "  |  " + strings.Join(warnings, "; ")
	}
	w.ProcSetWindowTextW.Call(installBtn, uintptr(unsafe.Pointer(w.UTF16("Snapvera"))))
	w.ProcSetWindowTextW.Call(cancelBtn, uintptr(unsafe.Pointer(w.UTF16("Close"))))
	w.ProcEnableWindow.Call(installBtn, 1)
	w.ProcEnableWindow.Call(startupBtn, 0)
	w.ProcInvalidateRect.Call(uintptr(win), 0, 1)
	if err := launch(); err != nil {
		statusText += "  |  Launch: " + err.Error()
		w.ProcInvalidateRect.Call(uintptr(win), 0, 1)
	}
}
func launch() error {
	if appPath == "" {
		return fmt.Errorf("application path is unavailable")
	}
	p, err := os.StartProcess(appPath, []string{appPath, "--background"}, &os.ProcAttr{Files: []*os.File{nil, nil, nil}})
	if err != nil {
		return err
	}
	if p != nil {
		_ = p.Release()
	}
	return nil
}

func drawInstallIcon(dc uintptr, r w.RECT) {
	cx := int32((int64(r.Left) + int64(r.Right)) / 2)
	cy := int32((int64(r.Top) + int64(r.Bottom)) / 2)
	pen, _, _ := w.ProcCreatePen.Call(w.PS_SOLID, 2, w.RGB(255, 255, 255))
	old, _, _ := w.ProcSelectObject.Call(dc, pen)
	w.ProcMoveToEx.Call(dc, uintptr(cx-6), uintptr(cy-3), 0)
	w.ProcLineTo.Call(dc, uintptr(cx), uintptr(cy+3))
	w.ProcLineTo.Call(dc, uintptr(cx+7), uintptr(cy-6))
	w.ProcSelectObject.Call(dc, old)
	w.ProcDeleteObject.Call(pen)
}
func drawButton(lp uintptr) uintptr {
	if lp == 0 {
		return 0
	}
	di := (*w.DRAWITEMSTRUCT)(unsafe.Pointer(lp))
	r := di.RcItem
	primary := int(di.CtlID) == idInstall
	disabled := di.ItemState&w.ODS_DISABLED != 0
	pressed := di.ItemState&w.ODS_SELECTED != 0
	focused := di.ItemState&w.ODS_FOCUS != 0
	bg := w.RGB(24, 33, 49)
	fg := w.RGB(230, 237, 249)
	stroke := w.RGB(49, 63, 87)
	iconBg := w.RGB(38, 50, 70)
	if primary {
		bg, fg, stroke, iconBg = w.RGB(96, 86, 255), w.RGB(255, 255, 255), w.RGB(128, 120, 255), w.RGB(73, 65, 215)
	}
	if pressed {
		if primary {
			bg = w.RGB(79, 68, 225)
		} else {
			bg = w.RGB(32, 42, 60)
		}
	}
	if disabled {
		fg = w.RGB(112, 123, 142)
		bg = w.RGB(21, 27, 39)
		stroke = w.RGB(39, 48, 64)
	}
	brush, _, _ := w.ProcCreateSolidBrush.Call(bg)
	pen, _, _ := w.ProcCreatePen.Call(w.PS_SOLID, 1, stroke)
	oldB, _, _ := w.ProcSelectObject.Call(di.HDC, brush)
	oldP, _, _ := w.ProcSelectObject.Call(di.HDC, pen)
	w.ProcRoundRect.Call(di.HDC, uintptr(r.Left+1), uintptr(r.Top+1), uintptr(r.Right-1), uintptr(r.Bottom-1), 18, 18)
	w.ProcSelectObject.Call(di.HDC, oldP)
	w.ProcSelectObject.Call(di.HDC, oldB)
	w.ProcDeleteObject.Call(pen)
	w.ProcDeleteObject.Call(brush)
	iconR := w.RECT{Left: r.Left + 10, Top: r.Top + 10, Right: r.Left + 46, Bottom: r.Bottom - 10}
	ib, _, _ := w.ProcCreateSolidBrush.Call(iconBg)
	oldB, _, _ = w.ProcSelectObject.Call(di.HDC, ib)
	w.ProcRoundRect.Call(di.HDC, uintptr(iconR.Left), uintptr(iconR.Top), uintptr(iconR.Right), uintptr(iconR.Bottom), 12, 12)
	w.ProcSelectObject.Call(di.HDC, oldB)
	w.ProcDeleteObject.Call(ib)
	if primary {
		drawInstallIcon(di.HDC, iconR)
	} else {
		p, _, _ := w.ProcCreatePen.Call(w.PS_SOLID, 2, fg)
		o, _, _ := w.ProcSelectObject.Call(di.HDC, p)
		w.ProcMoveToEx.Call(di.HDC, uintptr(iconR.Left+11), uintptr(iconR.Top+10), 0)
		w.ProcLineTo.Call(di.HDC, uintptr(iconR.Right-10), uintptr(iconR.Bottom-9))
		w.ProcMoveToEx.Call(di.HDC, uintptr(iconR.Right-10), uintptr(iconR.Top+10), 0)
		w.ProcLineTo.Call(di.HDC, uintptr(iconR.Left+11), uintptr(iconR.Bottom-9))
		w.ProcSelectObject.Call(di.HDC, o)
		w.ProcDeleteObject.Call(p)
	}
	var b [256]uint16
	w.ProcGetWindowTextW.Call(di.HwndItem, uintptr(unsafe.Pointer(&b[0])), 256)
	text := syscall.UTF16ToString(b[:])
	w.ProcSetBkMode.Call(di.HDC, w.TRANSPARENT)
	w.ProcSetTextColor.Call(di.HDC, fg)
	w.ProcSelectObject.Call(di.HDC, uintptr(fontRegular))
	tr := w.RECT{Left: iconR.Right + 12, Top: r.Top + 7, Right: r.Right - 14, Bottom: r.Bottom - 7}
	w.ProcDrawTextW.Call(di.HDC, uintptr(unsafe.Pointer(w.UTF16(text))), uintptr(len([]rune(text))), uintptr(unsafe.Pointer(&tr)), w.DT_CENTER|w.DT_VCENTER|w.DT_WORDBREAK)
	if focused && !disabled {
		fr := r
		fr.Left += 4
		fr.Top += 4
		fr.Right -= 4
		fr.Bottom -= 4
		w.ProcDrawFocusRect.Call(di.HDC, uintptr(unsafe.Pointer(&fr)))
	}
	return 1
}

func paint(hwnd uintptr) {
	var ps w.PAINTSTRUCT
	dc, _, _ := w.ProcBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	var cr w.RECT
	w.ProcGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
	bg, _, _ := w.ProcCreateSolidBrush.Call(w.RGB(9, 13, 20))
	w.ProcFillRect.Call(dc, uintptr(unsafe.Pointer(&cr)), bg)
	w.ProcDeleteObject.Call(bg)
	card, _, _ := w.ProcCreateSolidBrush.Call(w.RGB(14, 20, 30))
	old, _, _ := w.ProcSelectObject.Call(dc, card)
	w.ProcRoundRect.Call(dc, 22, 86, uintptr(cr.Right-22), 352, 26, 26)
	w.ProcSelectObject.Call(dc, old)
	w.ProcDeleteObject.Call(card)
	w.ProcSetBkMode.Call(dc, w.TRANSPARENT)
	w.ProcSetTextColor.Call(dc, w.RGB(248, 250, 255))
	w.ProcSelectObject.Call(dc, uintptr(fontTitle))
	title := buildinfo.Name
	rt := w.RECT{Left: 32, Top: 24, Right: cr.Right - 32, Bottom: 60}
	w.ProcDrawTextW.Call(dc, uintptr(unsafe.Pointer(w.UTF16(title))), uintptr(len([]rune(title))), uintptr(unsafe.Pointer(&rt)), w.DT_LEFT|w.DT_VCENTER|w.DT_SINGLELINE)
	w.ProcSetTextColor.Call(dc, w.RGB(154, 170, 196))
	w.ProcSelectObject.Call(dc, uintptr(fontRegular))
	sub := fmt.Sprintf("%s %s  •  %s  •  Windows %s", t("version", "Version"), buildinfo.Version, buildinfo.Publisher, runtime.GOARCH)
	rs := w.RECT{Left: 32, Top: 58, Right: cr.Right - 32, Bottom: 82}
	w.ProcDrawTextW.Call(dc, uintptr(unsafe.Pointer(w.UTF16(sub))), uintptr(len([]rune(sub))), uintptr(unsafe.Pointer(&rs)), w.DT_LEFT|w.DT_VCENTER|w.DT_SINGLELINE)
	w.ProcSetTextColor.Call(dc, w.RGB(237, 241, 250))
	body := "Per-user installation — administrator rights are not required.\nInstall location: %LOCALAPPDATA%\\Programs\\Snapvera\n\n" + statusText
	rb := w.RECT{Left: 44, Top: 110, Right: cr.Right - 44, Bottom: 236}
	w.ProcDrawTextW.Call(dc, uintptr(unsafe.Pointer(w.UTF16(body))), uintptr(len([]rune(body))), uintptr(unsafe.Pointer(&rb)), w.DT_LEFT|w.DT_WORDBREAK)
	w.ProcSetTextColor.Call(dc, w.RGB(139, 154, 179))
	footer := buildinfo.Website
	rf := w.RECT{Left: 32, Top: cr.Bottom - 34, Right: cr.Right - 32, Bottom: cr.Bottom - 12}
	w.ProcDrawTextW.Call(dc, uintptr(unsafe.Pointer(w.UTF16(footer))), uintptr(len([]rune(footer))), uintptr(unsafe.Pointer(&rf)), w.DT_CENTER|w.DT_VCENTER|w.DT_SINGLELINE)
	w.ProcEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}
func wndProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case w.WM_CREATE:
		win = w.HWND(hwnd)
		ensureFonts()
		installBtn = button(hwnd, idInstall, t("install", "Install"), 44, 254, 344)
		cancelBtn = button(hwnd, idCancel, t("cancel", "Cancel"), 400, 254, 344)
		startupBtn = button(hwnd, idStartupToggle, "✓  "+t("startup", "Start with Windows"), 44, 318, 700)
		return 0
	case w.WM_DRAWITEM:
		return drawButton(lp)
	case w.WM_COMMAND:
		switch int(w.Loword(wp)) {
		case idInstall:
			if installed {
				if err := launch(); err != nil {
					statusText = "Launch failed: " + err.Error()
					w.ProcInvalidateRect.Call(hwnd, 0, 1)
				}
			} else {
				doInstall()
			}
		case idCancel:
			w.ProcPostMessageW.Call(hwnd, w.WM_CLOSE, 0, 0)
		case idStartupToggle:
			if !installed {
				startWithWindows = !startWithWindows
				prefix := "○  "
				if startWithWindows {
					prefix = "✓  "
				}
				w.ProcSetWindowTextW.Call(startupBtn, uintptr(unsafe.Pointer(w.UTF16(prefix+t("startup", "Start with Windows")))))
				w.ProcInvalidateRect.Call(startupBtn, 0, 1)
			}
		}
		return 0
	case w.WM_ERASEBKGND:
		return 1
	case w.WM_PAINT:
		paint(hwnd)
		return 0
	case w.WM_CLOSE:
		w.ProcDestroyWindow.Call(hwnd)
		return 0
	case w.WM_DESTROY:
		w.ProcPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := w.ProcDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}
func main() {
	initI18N()
	ensureFonts()
	appIcon = loadIcon()
	procCB = syscall.NewCallback(wndProc)
	inst, _, _ := w.ProcGetModuleHandleW.Call(0)
	cls := w.UTF16("Snapvera.Setup.Go.v100")
	wc := w.WNDCLASSEX{CbSize: uint32(unsafe.Sizeof(w.WNDCLASSEX{})), Style: w.CS_HREDRAW | w.CS_VREDRAW, LpfnWndProc: procCB, HInstance: w.HINSTANCE(inst), LpszClassName: cls, HIcon: appIcon, HIconSm: appIcon}
	cur, _, _ := w.ProcLoadCursorW.Call(0, w.IntResource(w.IDC_ARROW))
	wc.HCursor = w.HCURSOR(cur)
	if r, _, _ := w.ProcRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return
	}
	statusText = "Ready to install Snapvera " + buildinfo.Version
	h, _, _ := w.ProcCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(w.UTF16("Snapvera "+buildinfo.Version+" — Setup"))), w.WS_OVERLAPPED|w.WS_CAPTION|w.WS_SYSMENU|w.WS_MINIMIZEBOX, w.CW_USEDEFAULT, w.CW_USEDEFAULT, 790, 458, 0, 0, inst, 0)
	if h == 0 {
		return
	}
	w.ProcDwmSetWindowAttribute.Call(h, w.DWMWA_USE_IMMERSIVE_DARK_MODE, uintptr(unsafe.Pointer(&[]uint32{1}[0])), 4)
	corner := uint32(w.DWMWCP_ROUND)
	w.ProcDwmSetWindowAttribute.Call(h, w.DWMWA_WINDOW_CORNER_PREFERENCE, uintptr(unsafe.Pointer(&corner)), 4)
	w.ProcShowWindow.Call(h, w.SW_SHOW)
	w.ProcUpdateWindow.Call(h)
	var m w.MSG
	for {
		r, _, _ := w.ProcGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		w.ProcTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		w.ProcDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
}
