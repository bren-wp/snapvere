package win

// DWM frame bounds exclude the invisible resize border/shadow that
// GetWindowRect may include on modern Windows. Keep this optional API
// isolated so active-window capture can gracefully fall back on older systems.
const DWMWA_EXTENDED_FRAME_BOUNDS = 9

var ProcDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")
