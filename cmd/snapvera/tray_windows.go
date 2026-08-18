package main

import (
	"fmt"
	"time"
	"unsafe"

	w "snapvera/internal/win"
)

func rememberForeground() {
	fh, _, _ := w.ProcGetForegroundWindow.Call()
	if fh != 0 && w.HWND(fh) != mainHWND && w.HWND(fh) != settingsHWND {
		lastExternal = w.HWND(fh)
	}
}
func addTray() bool {
	if trayAdded {
		return true
	}
	var n w.NOTIFYICONDATA
	n.CbSize = uint32(unsafe.Sizeof(n))
	n.HWnd = mainHWND
	n.UID = 1
	n.UFlags = w.NIF_MESSAGE | w.NIF_ICON | w.NIF_TIP
	n.UCallbackMessage = wmTray
	n.HIcon = appIcon
	copyUTF16Fixed(n.SzTip[:], "Snapvera — "+t("tagline", "Screen capture"))
	r, _, _ := w.ProcShellNotifyIconW.Call(w.NIM_ADD, uintptr(unsafe.Pointer(&n)))
	trayAdded = r != 0
	return trayAdded
}
func removeTray() {
	if !trayAdded {
		return
	}
	var n w.NOTIFYICONDATA
	n.CbSize = uint32(unsafe.Sizeof(n))
	n.HWnd = mainHWND
	n.UID = 1
	w.ProcShellNotifyIconW.Call(w.NIM_DELETE, uintptr(unsafe.Pointer(&n)))
	trayAdded = false
}
func unregisterHotkeys() {
	for _, id := range []uintptr{3001, 3002, 3003, 3004, 3005} {
		w.ProcUnregisterHotKey.Call(uintptr(mainHWND), id)
	}
}
func registerHotkeys() {
	unregisterHotkeys()
	if !prefs.HotkeysEnabled {
		return
	}
	mods := uintptr(w.MOD_NOREPEAT)
	if r, _, _ := w.ProcRegisterHotKey.Call(uintptr(mainHWND), 3001, mods, w.VK_SNAPSHOT); r == 0 {
		w.ProcRegisterHotKey.Call(uintptr(mainHWND), 3001, w.MOD_CONTROL|w.MOD_SHIFT|w.MOD_NOREPEAT, w.VK_S)
	}
	w.ProcRegisterHotKey.Call(uintptr(mainHWND), 3002, w.MOD_CONTROL|w.MOD_SHIFT|w.MOD_NOREPEAT, w.VK_F)
	w.ProcRegisterHotKey.Call(uintptr(mainHWND), 3003, w.MOD_CONTROL|w.MOD_SHIFT|w.MOD_NOREPEAT, w.VK_W)
	w.ProcRegisterHotKey.Call(uintptr(mainHWND), 3004, w.MOD_CONTROL|w.MOD_SHIFT|w.MOD_NOREPEAT, w.VK_D)
	w.ProcRegisterHotKey.Call(uintptr(mainHWND), 3005, w.MOD_CONTROL|w.MOD_SHIFT|w.MOD_NOREPEAT, w.VK_R)
	logf("hotkeys registered enabled=%v", prefs.HotkeysEnabled)
}
func trayMenu() {
	rememberForeground()
	m, _, _ := w.ProcCreatePopupMenu.Call()
	if m == 0 {
		return
	}
	defer w.ProcDestroyMenu.Call(m)
	appendItem := func(id int, text string, flags uintptr) {
		var ptr uintptr
		if text != "" {
			ptr = uintptr(unsafe.Pointer(w.UTF16(text)))
		}
		w.ProcAppendMenuW.Call(m, flags, uintptr(id), ptr)
	}
	appendItem(idRegion, t("history_image", "Screenshot")+" — "+t("capture_area", "Capture area"), w.MF_STRING)
	appendItem(idWindow, t("history_image", "Screenshot")+" — "+t("capture_window", "Capture active window"), w.MF_STRING)
	appendItem(idFull, t("history_image", "Screenshot")+" — "+t("capture_full", "Capture full desktop"), w.MF_STRING)
	appendItem(idDelay, fmt.Sprintf("%s (%ds)", t("capture_delay_label", "Delayed capture"), prefs.DelaySeconds), w.MF_STRING)
	appendItem(0, "", w.MF_SEPARATOR)
	if recordingActive() {
		appendItem(idRecordStop, t("history_video", "Video")+" — "+t("recording_stop", "Stop recording")+"  ·  Ctrl+Shift+R", w.MF_STRING)
	} else {
		appendItem(idRecordFull, t("history_video", "Video")+" — "+t("recording_full", "Record full desktop"), w.MF_STRING)
		appendItem(idRecordRegion, t("history_video", "Video")+" — "+t("recording_region", "Record area"), w.MF_STRING)
	}
	appendItem(idOpenVideos, t("open_video_folder", "Open recording folder"), w.MF_STRING)
	appendItem(0, "", w.MF_SEPARATOR)
	appendItem(idSettings, t("settings", "Settings"), w.MF_STRING)
	appendItem(idOpen, t("open_folder", "Open screenshot folder"), w.MF_STRING)
	appendItem(idHistory, t("history", "History"), w.MF_STRING)
	appendItem(idDiag, t("diagnostics", "Diagnostics"), w.MF_STRING)
	appendItem(idAbout, t("about", "About Snapvera"), w.MF_STRING)
	appendItem(0, "", w.MF_SEPARATOR)
	appendItem(idExit, t("exit", "Exit"), w.MF_STRING)
	var pt w.POINT
	w.ProcGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	w.ProcSetForegroundWindow.Call(uintptr(mainHWND))
	cmd, _, _ := w.ProcTrackPopupMenu.Call(m, w.TPM_RIGHTBUTTON|w.TPM_RETURNCMD, uintptr(pt.X), uintptr(pt.Y), 0, uintptr(mainHWND), 0)
	if cmd != 0 {
		handleCommand(int(cmd))
	}
	w.ProcPostMessageW.Call(uintptr(mainHWND), w.WM_NULL, 0, 0)
}
func handleCommand(id int) {
	switch id {
	case idRegion:
		goCapture("region", 0)
	case idFull:
		goCapture("full", 0)
	case idWindow:
		goCapture("window", 0)
	case idDelay:
		goCapture("region", time.Duration(prefs.DelaySeconds)*time.Second)
	case idRecordFull:
		startRecording("full")
	case idRecordRegion:
		startRecording("region")
	case idRecordStop:
		stopRecording()
	case idOpenVideos:
		w.ProcShellExecuteW.Call(0, uintptr(unsafe.Pointer(w.UTF16("open"))), uintptr(unsafe.Pointer(w.UTF16(videoDir()))), 0, 0, w.SW_SHOWNORMAL)
	case idOpen:
		openFolder()
	case idSettings:
		showSettings()
	case idHistory:
		showHistory()
	case idAbout:
		showAbout()
	case idDiag:
		runDiagnosticsUI()
	case idExit:
		stopRecordingAndWait(3 * time.Second)
		w.ProcDestroyWindow.Call(uintptr(mainHWND))
	}
}
func goCapture(mode string, delay time.Duration) { doCapture(mode, delay) }
func hostProc(hwnd uintptr, msg uint32, wp, lp uintptr) uintptr {
	if taskbarCreatedMsg != 0 && msg == taskbarCreatedMsg {
		trayAdded = false
		addTray()
		return 0
	}
	switch msg {
	case wmRecordingState:
		refreshRecordingButtons()
		return 0
	case wmTray:
		event := uint32(lp)
		switch event {
		case w.WM_LBUTTONUP:
			rememberForeground()
			if prefs.TrayLeftClick == "region" {
				goCapture("region", 0)
			} else {
				showSettings()
			}
		case w.WM_LBUTTONDBLCLK:
			showSettings()
		case w.WM_RBUTTONUP, w.WM_CONTEXTMENU:
			trayMenu()
		}
		return 0
	case w.WM_HOTKEY:
		rememberForeground()
		switch wp {
		case 3001:
			goCapture("region", 0)
		case 3002:
			goCapture("full", 0)
		case 3003:
			goCapture("window", 0)
		case 3004:
			goCapture("region", time.Duration(prefs.DelaySeconds)*time.Second)
		case 3005:
			if recordingActive() {
				stopRecording()
			} else {
				startRecording("full")
			}
		}
		return 0
	case w.WM_CLOSE:
		stopRecordingAndWait(3 * time.Second)
		w.ProcDestroyWindow.Call(hwnd)
		return 0
	case w.WM_DESTROY:
		stopRecordingAndWait(3 * time.Second)
		closePinnedWindows()
		if historyHWND != 0 {
			w.ProcDestroyWindow.Call(uintptr(historyHWND))
		}
		unregisterHotkeys()
		removeTray()
		releaseFonts()
		w.ProcPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := w.ProcDefWindowProcW.Call(hwnd, uintptr(msg), wp, lp)
	return r
}
func refreshRecordingButtons() {
	active := recordingActive()
	if btnRecordFull != 0 {
		w.ProcEnableWindow.Call(btnRecordFull, boolToWin(!active))
	}
	if btnRecordRegion != 0 {
		w.ProcEnableWindow.Call(btnRecordRegion, boolToWin(!active))
	}
	if btnRecordStop != 0 {
		w.ProcEnableWindow.Call(btnRecordStop, boolToWin(active))
	}
	if settingsHWND != 0 {
		w.ProcInvalidateRect.Call(uintptr(settingsHWND), 0, 1)
	}
}
func boolToWin(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}

func optionText() {
	if btnHotkeys != 0 {
		w.ProcSetWindowTextW.Call(btnHotkeys, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("hotkeys", "Hotkeys"), boolWord(prefs.HotkeysEnabled))))))
	}
	if btnDelay != 0 {
		w.ProcSetWindowTextW.Call(btnDelay, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("capture_delay_label", "Capture delay"), fmt.Sprintf("%d s", prefs.DelaySeconds))))))
	}
	if btnTheme != 0 {
		mode := prefs.Theme
		switch prefs.Theme {
		case "system":
			mode = t("system", "System")
		case "dark":
			mode = t("dark", "Dark")
		case "light":
			mode = t("light", "Light")
		}
		w.ProcSetWindowTextW.Call(btnTheme, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("theme", "Theme"), mode)))))
	}
	if btnTrayClick != 0 {
		action := t("capture_area", "Capture area")
		if prefs.TrayLeftClick != "region" {
			action = t("settings", "Settings")
		}
		w.ProcSetWindowTextW.Call(btnTrayClick, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("tray_left_click", "Tray left click"), action)))))
	}
	if btnNotify != 0 {
		w.ProcSetWindowTextW.Call(btnNotify, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("notifications", "Notifications"), boolWord(prefs.NotifyErrors))))))
	}
	if btnStartup != 0 {
		value := boolWord(prefs.StartWithWindows)
		if isPortable() {
			value = t("installer_only", "Installer only")
		}
		label := optionLabel(t("start_windows", "Start with Windows"), value)
		w.ProcSetWindowTextW.Call(btnStartup, uintptr(unsafe.Pointer(w.UTF16(label))))
		enabled := uintptr(1)
		if isPortable() {
			enabled = 0
		}
		w.ProcEnableWindow.Call(btnStartup, enabled)
	}
	if btnExport != 0 {
		w.ProcSetWindowTextW.Call(btnExport, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("export_preset", "Image export"), exportPresetLabel())))))
	}
	if btnNaming != 0 {
		w.ProcSetWindowTextW.Call(btnNaming, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("naming_preset", "File naming"), namingPresetLabel())))))
	}
	if btnRecording != 0 {
		w.ProcSetWindowTextW.Call(btnRecording, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("recording_preset", "Video preset"), recordingPresetLabel())))))
	}
	if btnOCRLanguage != 0 {
		w.ProcSetWindowTextW.Call(btnOCRLanguage, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("ocr_language", "OCR language"), ocrLanguageLabel())))))
	}
	if btnHistoryToggle != 0 {
		w.ProcSetWindowTextW.Call(btnHistoryToggle, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("history_enabled", "History"), historyEnabledLabel())))))
	}
	if btnHistoryLimit != 0 {
		w.ProcSetWindowTextW.Call(btnHistoryLimit, uintptr(unsafe.Pointer(w.UTF16(optionLabel(t("history_limit", "History limit"), historyLimitLabel())))))
	}
	refreshRecordingButtons()
}
func cycleDelay() {
	switch prefs.DelaySeconds {
	case 0:
		prefs.DelaySeconds = 3
	case 3:
		prefs.DelaySeconds = 5
	case 5:
		prefs.DelaySeconds = 10
	default:
		prefs.DelaySeconds = 0
	}
	savePrefs()
	optionText()
}
func cycleTheme() {
	if prefs.Theme == "system" {
		prefs.Theme = "dark"
	} else if prefs.Theme == "dark" {
		prefs.Theme = "light"
	} else {
		prefs.Theme = "system"
	}
	savePrefs()
	optionText()
	if settingsHWND != 0 {
		applyWindowStyle(uintptr(settingsHWND))
		w.ProcInvalidateRect.Call(uintptr(settingsHWND), 0, 1)
	}
}
func cycleExportPreset() {
	switch prefs.ExportPreset {
	case "png":
		prefs.ExportPreset = "jpg-high"
	case "jpg-high":
		prefs.ExportPreset = "jpg-balanced"
	case "jpg-balanced":
		prefs.ExportPreset = "jpg-small"
	default:
		prefs.ExportPreset = "png"
	}
	savePrefs()
	optionText()
}
func cycleNamePreset() {
	switch prefs.NamePreset {
	case "standard":
		prefs.NamePreset = "compact"
	case "compact":
		prefs.NamePreset = "timestamp"
	case "timestamp":
		prefs.NamePreset = "technical"
	default:
		prefs.NamePreset = "standard"
	}
	savePrefs()
	optionText()
}
func cycleRecordingPreset() {
	switch prefs.RecordingPreset {
	case "compact":
		prefs.RecordingPreset = "balanced"
	case "balanced":
		prefs.RecordingPreset = "smooth"
	default:
		prefs.RecordingPreset = "compact"
	}
	savePrefs()
	optionText()
}
func cycleOCRLanguage() {
	opts := ocrLanguageOptions()
	idx := 0
	for i, v := range opts {
		if v == prefs.OCRLanguage {
			idx = (i + 1) % len(opts)
			break
		}
	}
	prefs.OCRLanguage = opts[idx]
	savePrefs()
	optionText()
}
