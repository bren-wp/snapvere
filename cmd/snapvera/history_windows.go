package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	hist "snapvera/internal/history"
	w "snapvera/internal/win"
)

const (
	idHistoryItemBase = 5000
	historyPageSize   = 8
	wmHistoryRefresh  = w.WM_APP + 44
)

var (
	historyMu      sync.Mutex
	historyHWND    w.HWND
	historyProcCB  uintptr
	historyButtons []uintptr
	historyEntries []hist.Entry
	historyPage    int
)

func historyIndexFile() string {
	return filepath.Join(appDataDir(), "history.json")
}

func addHistoryEntry(path, kind string, width, height int32) {
	if !prefs.HistoryEnabled || path == "" {
		return
	}
	e := hist.Entry{Path: path, Kind: kind, Created: time.Now(), Width: width, Height: height}
	historyMu.Lock()
	err := hist.Add(historyIndexFile(), e, prefs.HistoryLimit)
	historyMu.Unlock()
	if err != nil {
		logf("history add failed path=%s err=%v", path, err)
		return
	}
	if historyHWND != 0 {
		w.ProcPostMessageW.Call(uintptr(historyHWND), wmHistoryRefresh, 0, 0)
	}
}

func historyEnabledLabel() string { return boolWord(prefs.HistoryEnabled) }
func historyLimitLabel() string   { return fmt.Sprintf("%d", prefs.HistoryLimit) }

func cycleHistoryLimit() {
	switch prefs.HistoryLimit {
	case 50:
		prefs.HistoryLimit = 100
	case 100:
		prefs.HistoryLimit = 250
	case 250:
		prefs.HistoryLimit = 500
	default:
		prefs.HistoryLimit = 50
	}
	savePrefs()
	optionText()
	if historyHWND != 0 {
		refreshHistoryWindow()
	}
}

func toggleHistory() {
	prefs.HistoryEnabled = !prefs.HistoryEnabled
	savePrefs()
	optionText()
}

func loadHistoryForUI() []hist.Entry {
	historyMu.Lock()
	defer historyMu.Unlock()
	entries, err := hist.RemoveMissing(historyIndexFile(), prefs.HistoryLimit)
	if err != nil {
		logf("history load failed: %v", err)
		return nil
	}
	return entries
}

func destroyHistoryButtons() {
	for _, h := range historyButtons {
		if h != 0 {
			w.ProcDestroyWindow.Call(h)
		}
	}
	historyButtons = historyButtons[:0]
}

func historyKindLabel(kind string) string {
	if kind == "video" {
		return t("history_video", "Video")
	}
	return t("history_image", "Screenshot")
}

func humanBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	if n < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(n)/(1024*1024*1024))
}

func historyItemLabel(e hist.Entry) string {
	dims := ""
	if e.Width > 0 && e.Height > 0 {
		dims = fmt.Sprintf(" · %d×%d", e.Width, e.Height)
	}
	size := ""
	if e.Size > 0 {
		size = " · " + humanBytes(e.Size)
	}
	return fmt.Sprintf("%s\n%s · %s%s%s", filepath.Base(e.Path), historyKindLabel(e.Kind), e.Created.Local().Format("02.01.2006 15:04"), dims, size)
}

func refreshHistoryWindow() {
	if historyHWND == 0 {
		return
	}
	historyEntries = loadHistoryForUI()
	maxPage := 0
	if len(historyEntries) > 0 {
		maxPage = (len(historyEntries) - 1) / historyPageSize
	}
	if historyPage > maxPage {
		historyPage = maxPage
	}
	if historyPage < 0 {
		historyPage = 0
	}
	destroyHistoryButtons()
	start := historyPage * historyPageSize
	end := start + historyPageSize
	if end > len(historyEntries) {
		end = len(historyEntries)
	}
	y := int32(102)
	for i := start; i < end; i++ {
		id := idHistoryItemBase + (i - start)
		h := addModernButton(uintptr(historyHWND), id, historyItemLabel(historyEntries[i]), 28, y, 804, 58)
		historyButtons = append(historyButtons, h)
		y += 64
	}
	w.ProcInvalidateRect.Call(uintptr(historyHWND), 0, 1)
}

func openHistoryEntry(index int) {
	if index < 0 || index >= len(historyEntries) {
		return
	}
	p := historyEntries[index].Path
	if _, err := os.Stat(p); err != nil {
		refreshHistoryWindow()
		return
	}
	w.ProcShellExecuteW.Call(0, uintptr(unsafe.Pointer(w.UTF16("open"))), uintptr(unsafe.Pointer(w.UTF16(p))), 0, 0, w.SW_SHOWNORMAL)
}

func historyProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	switch msg {
	case w.WM_DRAWITEM:
		return drawModernButton(lp)
	case wmHistoryRefresh:
		refreshHistoryWindow()
		return 0
	case w.WM_COMMAND:
		id := int(w.Loword(wp))
		switch {
		case id >= idHistoryItemBase && id < idHistoryItemBase+historyPageSize:
			openHistoryEntry(historyPage*historyPageSize + id - idHistoryItemBase)
		case id == idHistoryPrev:
			if historyPage > 0 {
				historyPage--
				refreshHistoryWindow()
			}
		case id == idHistoryNext:
			if (historyPage+1)*historyPageSize < len(historyEntries) {
				historyPage++
				refreshHistoryWindow()
			}
		case id == idHistoryOpenPictures:
			w.ProcShellExecuteW.Call(0, uintptr(unsafe.Pointer(w.UTF16("open"))), uintptr(unsafe.Pointer(w.UTF16(pictureDir()))), 0, 0, w.SW_SHOWNORMAL)
		case id == idHistoryOpenVideos:
			w.ProcShellExecuteW.Call(0, uintptr(unsafe.Pointer(w.UTF16("open"))), uintptr(unsafe.Pointer(w.UTF16(videoDir()))), 0, 0, w.SW_SHOWNORMAL)
		}
		return 0
	case w.WM_PAINT:
		var ps w.PAINTSTRUCT
		dc, _, _ := w.ProcBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		var cr w.RECT
		w.ProcGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&cr)))
		dark := darkUI()
		bg := w.RGB(244, 247, 252)
		fg := w.RGB(22, 29, 42)
		muted := w.RGB(95, 108, 128)
		if dark {
			bg = w.RGB(9, 13, 20)
			fg = w.RGB(244, 247, 253)
			muted = w.RGB(154, 170, 196)
		}
		br, _, _ := w.ProcCreateSolidBrush.Call(bg)
		w.ProcFillRect.Call(dc, uintptr(unsafe.Pointer(&cr)), br)
		w.ProcDeleteObject.Call(br)
		drawCard(dc, w.RECT{Left: 18, Top: 86, Right: int32(cr.Right) - 18, Bottom: int32(cr.Bottom) - 70}, dark)
		w.ProcSetBkMode.Call(dc, w.TRANSPARENT)
		w.ProcSetTextColor.Call(dc, fg)
		w.ProcSelectObject.Call(dc, uintptr(fontTitle))
		title := t("history", "History")
		rt := w.RECT{Left: 28, Top: 18, Right: int32(cr.Right) - 28, Bottom: 52}
		w.ProcDrawTextW.Call(dc, uintptr(unsafe.Pointer(w.UTF16(title))), uintptr(len([]rune(title))), uintptr(unsafe.Pointer(&rt)), w.DT_LEFT|w.DT_VCENTER|w.DT_SINGLELINE)
		w.ProcSetTextColor.Call(dc, muted)
		w.ProcSelectObject.Call(dc, uintptr(fontRegular))
		countText := fmt.Sprintf("%s: %d", t("history_recent", "Recent local items"), len(historyEntries))
		rp := w.RECT{Left: 28, Top: 52, Right: int32(cr.Right) - 28, Bottom: 80}
		w.ProcDrawTextW.Call(dc, uintptr(unsafe.Pointer(w.UTF16(countText))), uintptr(len([]rune(countText))), uintptr(unsafe.Pointer(&rp)), w.DT_LEFT|w.DT_VCENTER|w.DT_SINGLELINE)
		if len(historyEntries) == 0 {
			empty := t("history_empty", "No saved screenshots or recordings yet.")
			re := w.RECT{Left: 52, Top: 150, Right: int32(cr.Right) - 52, Bottom: 240}
			w.ProcDrawTextW.Call(dc, uintptr(unsafe.Pointer(w.UTF16(empty))), uintptr(len([]rune(empty))), uintptr(unsafe.Pointer(&re)), w.DT_CENTER|w.DT_VCENTER|w.DT_WORDBREAK)
		}
		pages := 1
		if len(historyEntries) > 0 {
			pages = (len(historyEntries) + historyPageSize - 1) / historyPageSize
		}
		pageText := fmt.Sprintf("%s %d / %d", t("history_page", "Page"), historyPage+1, pages)
		rpage := w.RECT{Left: 330, Top: int32(cr.Bottom) - 58, Right: int32(cr.Right) - 330, Bottom: int32(cr.Bottom) - 18}
		w.ProcDrawTextW.Call(dc, uintptr(unsafe.Pointer(w.UTF16(pageText))), uintptr(len([]rune(pageText))), uintptr(unsafe.Pointer(&rpage)), w.DT_CENTER|w.DT_VCENTER|w.DT_SINGLELINE)
		w.ProcEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case w.WM_CLOSE:
		w.ProcShowWindow.Call(hwnd, w.SW_HIDE)
		return 0
	case w.WM_DESTROY:
		historyHWND = 0
		destroyHistoryButtons()
		return 0
	}
	r, _, _ := w.ProcDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}

func createHistoryWindow() w.HWND {
	inst, _, _ := w.ProcGetModuleHandleW.Call(0)
	cls := w.UTF16("Snapvera.History.Go.v1")
	wc := w.WNDCLASSEX{CbSize: uint32(unsafe.Sizeof(w.WNDCLASSEX{})), Style: w.CS_HREDRAW | w.CS_VREDRAW, LpfnWndProc: historyProcCB, HInstance: w.HINSTANCE(inst), LpszClassName: cls, HIcon: appIcon, HIconSm: appIcon}
	cur, _, _ := w.ProcLoadCursorW.Call(0, w.IntResource(w.IDC_ARROW))
	wc.HCursor = w.HCURSOR(cur)
	w.ProcRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	v := virtualDesktop()
	ww, hh := int32(860), int32(690)
	if ww > v.W-30 {
		ww = v.W - 30
	}
	if hh > v.H-30 {
		hh = v.H - 30
	}
	h, _, e := w.ProcCreateWindowExW.Call(0, uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(w.UTF16("Snapvera — "+t("history", "History")))), w.WS_OVERLAPPEDWINDOW, uintptr(v.X+(v.W-ww)/2), uintptr(v.Y+(v.H-hh)/2), uintptr(ww), uintptr(hh), 0, 0, inst, 0)
	if h == 0 {
		logf("history window create failed: %v", e)
		return 0
	}
	applyWindowStyle(h)
	addModernButton(h, idHistoryPrev, t("previous", "Previous"), 28, hh-92, 190, 46)
	addModernButton(h, idHistoryOpenPictures, t("open_folder", "Open screenshot folder"), 226, hh-92, 198, 46)
	addModernButton(h, idHistoryOpenVideos, t("open_video_folder", "Open recording folder"), 432, hh-92, 198, 46)
	addModernButton(h, idHistoryNext, t("next", "Next"), 638, hh-92, 190, 46)
	return w.HWND(h)
}

func showHistory() {
	if historyHWND == 0 {
		historyHWND = createHistoryWindow()
	}
	if historyHWND == 0 {
		return
	}
	refreshHistoryWindow()
	w.ProcShowWindow.Call(uintptr(historyHWND), w.SW_RESTORE)
	w.ProcShowWindow.Call(uintptr(historyHWND), w.SW_SHOW)
	w.ProcSetForegroundWindow.Call(uintptr(historyHWND))
}

func initHistoryCallback() { historyProcCB = syscall.NewCallback(historyProc) }
