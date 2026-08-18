package main

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"unsafe"

	"snapvera/internal/resources"
	w "snapvera/internal/win"
)

const maxPixels int64 = 20_000_000

var embeddedIcon = resources.Icon
var tr map[string]string
var currentLang = "en"

var mainHWND w.HWND
var appIcon w.HICON
var lastExternal w.HWND
var logPath string
var captureBusy bool
var settingsHWND w.HWND
var taskbarCreatedMsg uint32
var trayAdded bool
var singleMutex uintptr
var fontRegular w.HFONT
var fontTitle w.HFONT
var fontSemibold w.HFONT
var fontSmall w.HFONT

const (
	wmTray           = w.WM_APP + 20
	wmRecordingState = w.WM_APP + 21
)

type AppSettings struct {
	SchemaVersion    int    `json:"schema_version"`
	HotkeysEnabled   bool   `json:"hotkeys_enabled"`
	DelaySeconds     int    `json:"delay_seconds"`
	StartWithWindows bool   `json:"start_with_windows"`
	Theme            string `json:"theme"`
	TrayLeftClick    string `json:"tray_left_click"`
	NotifyErrors     bool   `json:"notify_errors"`
	ExportPreset     string `json:"export_preset"`
	NamePreset       string `json:"name_preset"`
	RecordingPreset  string `json:"recording_preset"`
	OCRLanguage      string `json:"ocr_language"`
	HistoryEnabled   bool   `json:"history_enabled"`
	HistoryLimit     int    `json:"history_limit"`
}

func defaultSettings() AppSettings {
	return AppSettings{SchemaVersion: 1, HotkeysEnabled: true, DelaySeconds: 3, Theme: "dark", TrayLeftClick: "region", NotifyErrors: true, ExportPreset: "png", NamePreset: "standard", RecordingPreset: "balanced", OCRLanguage: "auto", HistoryEnabled: true, HistoryLimit: 100}
}

var prefs = defaultSettings()
var btnStartup, btnHotkeys, btnDelay, btnTheme, btnTrayClick, btnNotify, btnExport, btnNaming, btnRecording, btnOCRLanguage, btnHistoryToggle, btnHistoryLimit uintptr
var btnRecordFull, btnRecordRegion, btnRecordStop uintptr

const (
	idRegion = 1001
	idFull = 1002
	idWindow = 1003
	idDelay = 1004
	idOpen = 1005
	idAbout = 1006
	idDiag = 1007
	idExit = 1008
	idSettings = 1009
	idStartupToggle = 1010
	idHotkeysToggle = 1011
	idDelayCycle = 1012
	idThemeCycle = 1013
	idWebsite = 1014
	idTrayClickCycle = 1015
	idNotifyToggle = 1016
	idRecordFull = 1017
	idRecordRegion = 1018
	idRecordStop = 1019
	idExportCycle = 1020
	idNameCycle = 1021
	idRecordingCycle = 1022
	idOCRLanguageCycle = 1023
	idOpenVideos = 1024
	idHistory = 1025
	idHistoryToggle = 1026
	idHistoryLimit = 1027
	idHistoryPrev = 1028
	idHistoryNext = 1029
	idHistoryOpenPictures = 1030
	idHistoryOpenVideos = 1031
)

func runTrayUI() int {
	if appIcon == 0 { appIcon = loadAppIcon() }
	mainHWND = createHost()
	if mainHWND == 0 { return 2 }
	taskbarCreatedMsg = uint32(func() uintptr { r, _, _ := w.ProcRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(w.UTF16("TaskbarCreated")))); return r }())
	if !addTray() { logf("tray add failed") }
	registerHotkeys()
	var m w.MSG
	for { r, _, _ := w.ProcGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0); if int32(r) <= 0 { break }; w.ProcTranslateMessage.Call(uintptr(unsafe.Pointer(&m))); w.ProcDispatchMessageW.Call(uintptr(unsafe.Pointer(&m))) }
	return 0
}

func selfTest() int {
	v := virtualDesktop(); if !validRect(v) { logf("selftest invalid desktop %+v", v); return 10 }
	r := Rect{v.X, v.Y, min32(64, v.W), min32(64, v.H)}
	c, err := captureRect(r); if err != nil { logf("selftest capture err %v", err); return 11 }; defer c.Close()
	p := filepath.Join(os.TempDir(), "Snapvera-self-test.png"); if err = savePNG(c, p); err != nil { return 12 }; _ = os.Remove(p)
	if err = recordingSelfTest(); err != nil { logf("selftest recording err %v", err); return 13 }
	return 0
}

var buildMode = "portable"
func isPortable() bool { return buildMode == "portable" }
func runApplication() (exitCode int) {
	defer func(){ if r:=recover(); r!=nil { logf("panic: %v\n%s", r, debug.Stack()); exitCode=90 } }()
	runtime.LockOSThread()
	if r,_,_:=w.ProcSetProcessDpiAwarenessContext.Call(^uintptr(3)); r==0 { w.ProcSetProcessDPIAware.Call() }
	debug.SetGCPercent(90); debug.SetMemoryLimit(512<<20)
	initI18N(); initLog(); loadPrefs(); ensureFonts()
	name:=w.UTF16("Local\\Snapvera.Brendigo.SingleInstance"); h,_,_:=w.ProcCreateMutexW.Call(0,0,uintptr(unsafe.Pointer(name))); singleMutex=h
	if h!=0 && w.LastErrorCode()==w.ERROR_ALREADY_EXISTS { logf("second instance ignored"); return 0 }
	if h!=0 { defer w.ProcCloseHandle.Call(h) }
	overlayProcCB=syscall.NewCallback(overlayProc); editorProcCB=syscall.NewCallback(editorProc); textInputProcCB=syscall.NewCallback(textInputProc); hostProcCB=syscall.NewCallback(hostProc); settingsProcCB=syscall.NewCallback(settingsProc); initHistoryCallback(); initPinCallback()
	args:=strings.Join(os.Args[1:]," "); if strings.Contains(args,"--self-test") { return selfTest() }
	code:=runTrayUI(); if code!=0 { logf("Snapvera exited with code %d",code) }; return code
}
func main(){ os.Exit(runApplication()) }
