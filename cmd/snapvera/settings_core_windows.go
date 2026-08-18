package main

import (
	"encoding/json"
	"errors"
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
	w "snapvera/internal/win"
)

func appDataDir() string {
	if isPortable() { if exe,err:=os.Executable(); err==nil && strings.TrimSpace(exe)!="" { return filepath.Join(filepath.Dir(exe),"SnapveraData") } }
	if base,err:=os.UserConfigDir(); err==nil && strings.TrimSpace(base)!="" { return filepath.Join(base,"Snapvera") }
	if base:=strings.TrimSpace(os.Getenv("LOCALAPPDATA")); base!="" { return filepath.Join(base,"Snapvera") }
	if exe,err:=os.Executable(); err==nil && strings.TrimSpace(exe)!="" { return filepath.Join(filepath.Dir(exe),"SnapveraData") }
	return filepath.Join(os.TempDir(),"Snapvera")
}
func settingsFile() string { return filepath.Join(appDataDir(),"settings.json") }
func normalizePrefs(p *AppSettings){ if p.SchemaVersion<=0{p.SchemaVersion=1}; if p.DelaySeconds!=0&&p.DelaySeconds!=3&&p.DelaySeconds!=5&&p.DelaySeconds!=10{p.DelaySeconds=3}; if p.Theme!="system"&&p.Theme!="light"&&p.Theme!="dark"{p.Theme="dark"}; if p.TrayLeftClick!="region"&&p.TrayLeftClick!="settings"{p.TrayLeftClick="region"}; switch p.ExportPreset{case "png","jpg-high","jpg-balanced","jpg-small":default:p.ExportPreset="png"}; switch p.NamePreset{case "standard","compact","timestamp","technical":default:p.NamePreset="standard"}; switch p.RecordingPreset{case "compact","balanced","smooth":default:p.RecordingPreset="balanced"}; if strings.TrimSpace(p.OCRLanguage)==""{p.OCRLanguage="auto"}; if p.HistoryLimit!=50&&p.HistoryLimit!=100&&p.HistoryLimit!=250&&p.HistoryLimit!=500{p.HistoryLimit=100} }
func loadPrefs(){ p:=settingsFile(); candidate:=defaultSettings(); b,err:=os.ReadFile(p); if err==nil { if err=json.Unmarshal(b,&candidate);err!=nil{logf("settings decode failed, defaults restored: %v",err);_ = os.Rename(p,p+".corrupt-"+time.Now().Format("20060102-150405"));candidate=defaultSettings()} } else if !os.IsNotExist(err){logf("settings read failed: %v",err)}; normalizePrefs(&candidate);prefs=candidate }
func savePrefs(){p:=settingsFile();dir:=filepath.Dir(p);if err:=os.MkdirAll(dir,0700);err!=nil{logf("settings mkdir failed: %v",err);return};b,err:=json.MarshalIndent(prefs,"","  ");if err!=nil{logf("settings encode failed: %v",err);return};f,err:=os.CreateTemp(dir,".settings-*.tmp");if err!=nil{logf("settings temp failed: %v",err);return};tmp:=f.Name();ok:=false;defer func(){_ = f.Close();if !ok{_ = os.Remove(tmp)}}();if err=f.Chmod(0600);err==nil{_,err=f.Write(b)};if err==nil{err=f.Sync()};if closeErr:=f.Close();err==nil{err=closeErr};if err!=nil{logf("settings write failed: %v",err);return};r,_,moveErr:=w.ProcMoveFileExW.Call(uintptr(unsafe.Pointer(w.UTF16(tmp))),uintptr(unsafe.Pointer(w.UTF16(p))),w.MOVEFILE_REPLACE_EXISTING|w.MOVEFILE_WRITE_THROUGH);if r==0{logf("settings replace failed: %v",moveErr);return};ok=true}
func boolWord(v bool)string{if v{return t("on","On")};return t("off","Off")}
func ensureFonts(){if fontRegular==0{r,_,_:=w.ProcCreateFontW.Call(^uintptr(16),0,0,0,w.FW_NORMAL,0,0,0,w.DEFAULT_CHARSET,w.OUT_DEFAULT_PRECIS,w.CLIP_DEFAULT_PRECIS,w.CLEARTYPE_QUALITY,w.DEFAULT_PITCH|w.FF_DONTCARE,uintptr(unsafe.Pointer(w.UTF16("Segoe UI"))));fontRegular=w.HFONT(r)};if fontSmall==0{r,_,_:=w.ProcCreateFontW.Call(^uintptr(13),0,0,0,w.FW_NORMAL,0,0,0,w.DEFAULT_CHARSET,w.OUT_DEFAULT_PRECIS,w.CLIP_DEFAULT_PRECIS,w.CLEARTYPE_QUALITY,w.DEFAULT_PITCH|w.FF_DONTCARE,uintptr(unsafe.Pointer(w.UTF16("Segoe UI"))));fontSmall=w.HFONT(r)};if fontSemibold==0{r,_,_:=w.ProcCreateFontW.Call(^uintptr(16),0,0,0,w.FW_SEMIBOLD,0,0,0,w.DEFAULT_CHARSET,w.OUT_DEFAULT_PRECIS,w.CLIP_DEFAULT_PRECIS,w.CLEARTYPE_QUALITY,w.DEFAULT_PITCH|w.FF_DONTCARE,uintptr(unsafe.Pointer(w.UTF16("Segoe UI"))));fontSemibold=w.HFONT(r)};if fontTitle==0{r,_,_:=w.ProcCreateFontW.Call(^uintptr(26),0,0,0,w.FW_SEMIBOLD,0,0,0,w.DEFAULT_CHARSET,w.OUT_DEFAULT_PRECIS,w.CLIP_DEFAULT_PRECIS,w.CLEARTYPE_QUALITY,w.DEFAULT_PITCH|w.FF_DONTCARE,uintptr(unsafe.Pointer(w.UTF16("Segoe UI"))));fontTitle=w.HFONT(r)}}
func releaseFonts(){for _,f:=range []w.HFONT{fontRegular,fontSmall,fontSemibold,fontTitle}{if f!=0{w.ProcDeleteObject.Call(uintptr(f))}};fontRegular,fontSmall,fontSemibold,fontTitle=0,0,0,0}
func darkUI()bool{if prefs.Theme=="dark"{return true};if prefs.Theme=="light"{return false};return systemDarkMode()}
func systemDarkMode()bool{const key=`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`;var h uintptr;r,_,_:=w.ProcRegOpenKeyExW.Call(w.HKEYCurrentUser(),uintptr(unsafe.Pointer(w.UTF16(key))),0,w.KEY_QUERY_VALUE,uintptr(unsafe.Pointer(&h)));if r!=0||h==0{return false};defer w.ProcRegCloseKey.Call(h);var typ uint32;var val uint32=1;sz:=uint32(4);r,_,_=w.ProcRegQueryValueExW.Call(h,uintptr(unsafe.Pointer(w.UTF16("AppsUseLightTheme"))),0,uintptr(unsafe.Pointer(&typ)),uintptr(unsafe.Pointer(&val)),uintptr(unsafe.Pointer(&sz)));return r==0&&typ==w.REG_DWORD&&val==0}
func applyWindowStyle(hwnd uintptr){ensureFonts();corner:=int32(2);w.ProcDwmSetWindowAttribute.Call(hwnd,w.DWMWA_WINDOW_CORNER_PREFERENCE,uintptr(unsafe.Pointer(&corner)),4);v:=int32(0);if darkUI(){v=1};w.ProcDwmSetWindowAttribute.Call(hwnd,w.DWMWA_USE_IMMERSIVE_DARK_MODE,uintptr(unsafe.Pointer(&v)),4)}
func setStartup(enabled bool){if isPortable(){prefs.StartWithWindows=false;return};const runKey=`Software\Microsoft\Windows\CurrentVersion\Run`;var h uintptr;var disp uint32;r,_,_:=w.ProcRegCreateKeyExW.Call(w.HKEYCurrentUser(),uintptr(unsafe.Pointer(w.UTF16(runKey))),0,0,0,w.KEY_SET_VALUE,0,uintptr(unsafe.Pointer(&h)),uintptr(unsafe.Pointer(&disp)));if r!=0{return};defer w.ProcRegCloseKey.Call(h);name:=uintptr(unsafe.Pointer(w.UTF16("Snapvera")));if !enabled{w.ProcRegDeleteValueW.Call(h,name);return};exe,_:=os.Executable();value:=`"`+exe+`" --background`;u:=syscall.StringToUTF16(value);w.ProcRegSetValueExW.Call(h,name,0,w.REG_SZ,uintptr(unsafe.Pointer(&u[0])),uintptr(len(u)*2))}
func notify(title,body string,kind uint32){if !trayAdded||!prefs.NotifyErrors{return};var n w.NOTIFYICONDATA;n.CbSize=uint32(unsafe.Sizeof(n));n.HWnd=mainHWND;n.UID=1;n.UFlags=w.NIF_INFO;n.DwInfoFlags=kind;copy(n.SzInfoTitle[:],syscall.StringToUTF16(title));copy(n.SzInfo[:],syscall.StringToUTF16(body));w.ProcShellNotifyIconW.Call(w.NIM_MODIFY,uintptr(unsafe.Pointer(&n)))}
func t(key,fallback string)string{if v:=tr[key];v!=""{return v};return fallback}
func initI18N(){locale:="";var buf[85]uint16;if r,_,_:=w.ProcGetUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])),85);r!=0{locale=syscall.UTF16ToString(buf[:])};currentLang,tr=i18n.Resolve(locale)}
func initLog(){dir:=filepath.Join(appDataDir(),"logs");_ = os.MkdirAll(dir,0700);logPath=filepath.Join(dir,"snapvera.log");if st,err:=os.Stat(logPath);err==nil&&st.Size()>1024*1024{_ = os.Rename(logPath,logPath+".old")};logf("start version=%s mode=%s arch=%s",buildinfo.Version,buildMode,runtime.GOARCH)}
func logf(f string,a ...any){if logPath==""{return};h,err:=os.OpenFile(logPath,os.O_CREATE|os.O_WRONLY|os.O_APPEND,0600);if err!=nil{return};defer h.Close();_,_=fmt.Fprintf(h,"%s %s\n",time.Now().Format("2006-01-02 15:04:05.000"),fmt.Sprintf(f,a...))}
func showError(step string,code uint32){logf("ERROR step=%s win32=%d",step,code);body:=fmt.Sprintf("%s  (%s: %d)",t("capture_failed","Screen capture failed."),step,code);if trayAdded{notify("Snapvera",body,w.NIIF_ERROR);return};w.ProcMessageBoxW.Call(uintptr(mainHWND),uintptr(unsafe.Pointer(w.UTF16(body))),uintptr(unsafe.Pointer(w.UTF16("Snapvera"))),w.MB_OK|w.MB_ICONERROR)}
func errorCode(err error)uint32{var e syscall.Errno;if errors.As(err,&e){return uint32(e)};return w.LastErrorCode()}
func message(text string,icon uintptr){w.ProcMessageBoxW.Call(uintptr(mainHWND),uintptr(unsafe.Pointer(w.UTF16(text))),uintptr(unsafe.Pointer(w.UTF16("Snapvera"))),w.MB_OK|icon)}
