package main

import (
	"strings"
	"syscall"
	"unsafe"

	w "snapvera/internal/win"
)

const modernButtonHeight int32 = 46

func actionLabel(kind, text string) string {
	kind = strings.TrimSpace(kind)
	text = strings.TrimSpace(text)
	if kind == "" {
		return text
	}
	return strings.ToUpper(kind) + "\n" + text
}

func optionLabel(title, value string) string {
	title = strings.TrimSpace(title)
	value = strings.TrimSpace(value)
	if value == "" {
		return title
	}
	return title + "\n" + value
}

func isCaptureActionButton(id int) bool { return id == idRegion || id == idFull || id == idWindow || id == idDelay }
func isVideoActionButton(id int) bool { return id == idRecordFull || id == idRecordRegion || id == idRecordStop }

func addModernButton(parent uintptr, id int, text string, x, y, wid, hei int32) uintptr {
	inst, _, _ := w.ProcGetModuleHandleW.Call(0)
	if hei < modernButtonHeight { hei = modernButtonHeight }
	h, _, _ := w.ProcCreateWindowExW.Call(0, uintptr(unsafe.Pointer(w.UTF16("BUTTON"))), uintptr(unsafe.Pointer(w.UTF16(text))), w.WS_CHILD|w.WS_VISIBLE|w.WS_TABSTOP|w.BS_OWNERDRAW, uintptr(x), uintptr(y), uintptr(wid), uintptr(hei), parent, uintptr(id), inst, 0)
	ensureFonts(); w.ProcSendMessageW.Call(h, w.WM_SETFONT, uintptr(fontRegular), 1)
	return h
}

func buttonActive(id int) bool {
	switch id {
	case eidArrow: return editor.hwnd != 0 && editor.tool == toolArrow
	case eidRect: return editor.hwnd != 0 && editor.tool == toolRect
	case eidEllipse: return editor.hwnd != 0 && editor.tool == toolEllipse
	case eidLine: return editor.hwnd != 0 && editor.tool == toolLine
	case eidPen: return editor.hwnd != 0 && editor.tool == toolPen
	case eidMarker: return editor.hwnd != 0 && editor.tool == toolMarker
	case eidText: return editor.hwnd != 0 && editor.tool == toolText
	case eidPixelate: return editor.hwnd != 0 && editor.tool == toolPixelate
	case eidBlur: return editor.hwnd != 0 && editor.tool == toolBlur
	case eidRedact: return editor.hwnd != 0 && editor.tool == toolRedact
	case eidCrop: return editor.hwnd != 0 && editor.tool == toolCrop
	}
	return false
}
func isRecordingButton(id int) bool { return id == idRecordFull || id == idRecordRegion || id == idRecordStop || id == idRecordingCycle }
func isPrimaryButton(id int) bool { switch id { case idRegion,idFull,idWindow,idDelay,eidSave,eidCopy,eidOCR,eidPin: return true }; return false }
func isDangerButton(id int) bool { return id == eidDiscard || id == idExit || id == idRecordStop || id == idOCRClose }

func fillRoundedRect(dc uintptr, r w.RECT, radius int32, fill, border uintptr) {
	br, _, _ := w.ProcCreateSolidBrush.Call(fill); pen, _, _ := w.ProcCreatePen.Call(w.PS_SOLID, 1, border)
	ob, _, _ := w.ProcSelectObject.Call(dc, br); op, _, _ := w.ProcSelectObject.Call(dc, pen)
	w.ProcRoundRect.Call(dc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), uintptr(radius), uintptr(radius))
	w.ProcSelectObject.Call(dc, ob); w.ProcSelectObject.Call(dc, op); w.ProcDeleteObject.Call(br); w.ProcDeleteObject.Call(pen)
}

func drawModernButton(lp uintptr) uintptr {
	if lp == 0 { return 0 }
	di := (*w.DRAWITEMSTRUCT)(unsafe.Pointer(lp)); dc := uintptr(di.HDC); rc := di.RcItem; id := int(di.CtlID); dark := darkUI()
	bg:=w.RGB(250,251,254); fg:=w.RGB(21,28,41); muted:=w.RGB(94,106,126); border:=w.RGB(216,223,235); accent:=w.RGB(79,104,255); accentSoft:=w.RGB(236,239,255); record:=w.RGB(218,54,86); recordSoft:=w.RGB(255,238,243)
	if dark { bg=w.RGB(20,27,40); fg=w.RGB(244,247,253); muted=w.RGB(151,164,187); border=w.RGB(43,56,78); accent=w.RGB(124,143,255); accentSoft=w.RGB(30,40,68); record=w.RGB(255,92,118); recordSoft=w.RGB(61,29,42) }
	selected:=di.ItemState&w.ODS_SELECTED!=0; hot:=di.ItemState&w.ODS_HOTLIGHT!=0; focused:=di.ItemState&w.ODS_FOCUS!=0; active:=buttonActive(id); disabled:=di.ItemState&w.ODS_DISABLED!=0
	if hot&&!disabled { if dark {bg=w.RGB(27,36,52);border=w.RGB(62,78,106)} else {bg=w.RGB(244,247,253);border=w.RGB(198,208,225)} }
	if selected { if dark {bg=w.RGB(31,40,57)} else {bg=w.RGB(240,243,250)} }
	if isCaptureActionButton(id) { if dark {bg=w.RGB(22,31,50)} else {bg=w.RGB(246,248,255)}; border=w.RGB(66,82,127) }
	if active { bg=accentSoft; border=accent }
	if isVideoActionButton(id)||id==idRecordingCycle { if dark {bg=w.RGB(36,25,35)} else {bg=w.RGB(255,248,250)}; border=w.RGB(101,48,66); if id==idRecordStop&&recordingActive(){bg=recordSoft;border=record} }
	if isDangerButton(id)&&id!=idRecordStop { if dark {bg=w.RGB(45,27,34);border=w.RGB(96,49,62)} else {bg=w.RGB(255,245,247);border=w.RGB(241,194,203)} }
	if disabled { fg=muted; if dark {bg=w.RGB(17,22,31);border=w.RGB(35,44,59)} else {bg=w.RGB(246,247,250);border=w.RGB(229,232,238)} }
	if focused&&!disabled { border=accent }; fillRoundedRect(dc,rc,16,bg,border)
	iconColor:=fg; plate:=accentSoft; if isVideoActionButton(id)||id==idRecordingCycle {iconColor=record;plate=recordSoft} else if isDangerButton(id){iconColor=record}
	plateTop:=rc.Top+10; plateBottom:=rc.Bottom-10; if plateBottom-plateTop<34 {plateTop=rc.Top+7;plateBottom=rc.Bottom-7}; plateRect:=w.RECT{Left:rc.Left+11,Top:plateTop,Right:rc.Left+49,Bottom:plateBottom}; fillRoundedRect(dc,plateRect,11,plate,plate)
	iconX:=rc.Left+21; iconY:=(rc.Top+rc.Bottom)/2; drawButtonIcon(dc,id,iconX,iconY,func()uintptr{if isVideoActionButton(id)||id==idRecordingCycle{return record};if active||isPrimaryButton(id){return accent};return iconColor}())
	n,_,_:=w.ProcGetWindowTextLengthW.Call(uintptr(di.HwndItem)); buf:=make([]uint16,int(n)+2); if len(buf)>0 {w.ProcGetWindowTextW.Call(uintptr(di.HwndItem),uintptr(unsafe.Pointer(&buf[0])),uintptr(len(buf)))}; text:=strings.TrimSpace(syscall.UTF16ToString(buf)); w.ProcSetBkMode.Call(dc,w.TRANSPARENT); left:=rc.Left+61; right:=rc.Right-12; parts:=strings.SplitN(text,"\n",2)
	if len(parts)==2 { top:=strings.TrimSpace(parts[0]); bottom:=strings.TrimSpace(parts[1]); firstColor:=fg; secondColor:=muted; firstFont:=fontSemibold; secondFont:=fontSmall; if isCaptureActionButton(id){firstColor=accent;firstFont=fontSmall;secondColor=fg;secondFont=fontSemibold}else if isVideoActionButton(id){firstColor=record;firstFont=fontSmall;secondColor=fg;secondFont=fontSemibold}; w.ProcSetTextColor.Call(dc,firstColor);w.ProcSelectObject.Call(dc,uintptr(firstFont));r1:=w.RECT{Left:left,Top:rc.Top+8,Right:right,Bottom:rc.Top+29};w.ProcDrawTextW.Call(dc,uintptr(unsafe.Pointer(w.UTF16(top))),uintptr(len([]rune(top))),uintptr(unsafe.Pointer(&r1)),w.DT_LEFT|w.DT_VCENTER|w.DT_SINGLELINE);w.ProcSetTextColor.Call(dc,secondColor);w.ProcSelectObject.Call(dc,uintptr(secondFont));r2:=w.RECT{Left:left,Top:rc.Top+27,Right:right,Bottom:rc.Bottom-6};w.ProcDrawTextW.Call(dc,uintptr(unsafe.Pointer(w.UTF16(bottom))),uintptr(len([]rune(bottom))),uintptr(unsafe.Pointer(&r2)),w.DT_LEFT|w.DT_VCENTER|w.DT_WORDBREAK)
	} else { w.ProcSetTextColor.Call(dc,fg); if isPrimaryButton(id)||isDangerButton(id){w.ProcSelectObject.Call(dc,uintptr(fontSemibold))}else{w.ProcSelectObject.Call(dc,uintptr(fontRegular))};tr:=w.RECT{Left:left,Top:rc.Top+7,Right:right,Bottom:rc.Bottom-7};w.ProcDrawTextW.Call(dc,uintptr(unsafe.Pointer(w.UTF16(text))),uintptr(len([]rune(text))),uintptr(unsafe.Pointer(&tr)),w.DT_LEFT|w.DT_VCENTER|w.DT_WORDBREAK) }
	return 1
}

func drawButtonIcon(dc uintptr,id int,x,y int32,color uintptr){
	pen,_,_:=w.ProcCreatePen.Call(w.PS_SOLID,2,color);op,_,_:=w.ProcSelectObject.Call(dc,pen);defer func(){w.ProcSelectObject.Call(dc,op);w.ProcDeleteObject.Call(pen)}();line:=func(x1,y1,x2,y2 int32){w.ProcMoveToEx.Call(dc,uintptr(x1),uintptr(y1),0);w.ProcLineTo.Call(dc,uintptr(x2),uintptr(y2))};rect:=func(l,t,r,b int32){h,_,_:=w.ProcGetStockObject.Call(w.HOLLOW_BRUSH);ob,_,_:=w.ProcSelectObject.Call(dc,h);w.ProcRoundRect.Call(dc,uintptr(l),uintptr(t),uintptr(r),uintptr(b),4,4);w.ProcSelectObject.Call(dc,ob)};ellipse:=func(l,t,r,b int32){h,_,_:=w.ProcGetStockObject.Call(w.HOLLOW_BRUSH);ob,_,_:=w.ProcSelectObject.Call(dc,h);w.ProcEllipse.Call(dc,uintptr(l),uintptr(t),uintptr(r),uintptr(b));w.ProcSelectObject.Call(dc,ob)};filledEllipse:=func(l,t,r,b int32){br,_,_:=w.ProcCreateSolidBrush.Call(color);ob,_,_:=w.ProcSelectObject.Call(dc,br);w.ProcEllipse.Call(dc,uintptr(l),uintptr(t),uintptr(r),uintptr(b));w.ProcSelectObject.Call(dc,ob);w.ProcDeleteObject.Call(br)};filledRect:=func(l,t,r,b int32){br,_,_:=w.ProcCreateSolidBrush.Call(color);rc:=w.RECT{Left:l,Top:t,Right:r,Bottom:b};w.ProcFillRect.Call(dc,uintptr(unsafe.Pointer(&rc)),br);w.ProcDeleteObject.Call(br)}
	if id>=idHistoryItemBase&&id<idHistoryItemBase+historyPageSize{rect(x,y-7,x+17,y+7);line(x+3,y-3,x+14,y-3);line(x+3,y+1,x+12,y+1);line(x+3,y+5,x+9,y+5);return}
	switch id {
	case eidArrow:line(x,y+6,x+16,y-7);line(x+16,y-7,x+8,y-6);line(x+16,y-7,x+14,y+1)
	case eidRect,idRegion,eidCrop:rect(x,y-8,x+17,y+8)
	case eidEllipse:ellipse(x,y-8,x+17,y+8)
	case eidLine:line(x,y+7,x+17,y-7)
	case eidPen:line(x,y+7,x+15,y-8);line(x+2,y+8,x+6,y+6)
	case eidMarker:rect(x+2,y-7,x+14,y+5);line(x,y+8,x+17,y+8)
	case eidText:line(x,y-7,x+17,y-7);line(x+8,y-7,x+8,y+8)
	case eidPixelate:rect(x,y-7,x+6,y-1);rect(x+9,y-7,x+16,y-1);rect(x,y+2,x+6,y+8);rect(x+9,y+2,x+16,y+8)
	case eidBlur:ellipse(x+1,y-7,x+15,y+7);ellipse(x+5,y-3,x+12,y+4)
	case eidRedact:br,_,_:=w.ProcCreateSolidBrush.Call(color);r:=w.RECT{Left:x,Top:y-6,Right:x+17,Bottom:y+7};w.ProcFillRect.Call(dc,uintptr(unsafe.Pointer(&r)),br);w.ProcDeleteObject.Call(br)
	case eidUndo:line(x+4,y-5,x,y);line(x,y,x+6,y+1);line(x+1,y,x+15,y+6)
	case eidRedo:line(x+12,y-5,x+17,y);line(x+17,y,x+11,y+1);line(x+16,y,x+2,y+6)
	case eidColor:br,_,_:=w.ProcCreateSolidBrush.Call(editor.color);ob,_,_:=w.ProcSelectObject.Call(dc,br);ellipse(x+1,y-7,x+15,y+7);w.ProcSelectObject.Call(dc,ob);w.ProcDeleteObject.Call(br)
	case eidWidth:line(x,y-6,x+17,y-6);line(x,y,x+17,y);line(x,y+7,x+17,y+7)
	case eidOCR:rect(x,y-7,x+17,y+7);line(x+4,y-3,x+13,y-3);line(x+4,y+2,x+13,y+2)
	case eidCopy:rect(x+4,y-6,x+16,y+7);rect(x,y-9,x+12,y+4)
	case eidSave,eidSaveAs:rect(x,y-8,x+17,y+8);line(x+4,y-4,x+13,y-4);line(x+5,y+2,x+12,y+2)
	case eidPin:line(x+3,y-7,x+14,y-7);line(x+6,y-7,x+6,y-1);line(x+11,y-7,x+11,y-1);line(x+3,y-1,x+14,y-1);line(x+8,y-1,x+8,y+8)
	case eidRetake:ellipse(x+1,y-7,x+16,y+8);line(x+1,y-7,x+1,y);line(x+1,y-7,x+7,y-7)
	case eidDiscard,idExit,idOCRClose:line(x+2,y-6,x+15,y+7);line(x+15,y-6,x+2,y+7)
	case idRecordFull:rect(x,y-8,x+18,y+8);filledEllipse(x+6,y-3,x+12,y+3)
	case idRecordRegion:line(x,y-7,x+6,y-7);line(x,y-7,x,y-1);line(x+18,y-7,x+12,y-7);line(x+18,y-7,x+18,y-1);line(x,y+7,x+6,y+7);line(x,y+7,x,y+1);line(x+18,y+7,x+12,y+7);line(x+18,y+7,x+18,y+1);filledEllipse(x+7,y-2,x+12,y+3)
	case idRecordStop:filledRect(x+4,y-5,x+14,y+5)
	case idRecordingCycle:rect(x,y-7,x+18,y+7);filledEllipse(x+6,y-3,x+12,y+3)
	case idFull:rect(x,y-7,x+17,y+7);line(x+4,y-3,x+13,y-3)
	case idWindow:rect(x,y-7,x+17,y+7);line(x,y-3,x+17,y-3)
	case idDelay,idDelayCycle:ellipse(x+1,y-7,x+15,y+7);line(x+8,y,x+8,y-5);line(x+8,y,x+12,y+3)
	case idSettings:ellipse(x+3,y-5,x+14,y+6);ellipse(x+7,y-1,x+10,y+2)
	case idHotkeysToggle:rect(x,y-7,x+17,y+7);line(x+3,y-3,x+5,y-3);line(x+8,y-3,x+10,y-3);line(x+13,y-3,x+15,y-3);line(x+3,y+2,x+15,y+2)
	case idStartupToggle:line(x+8,y+8,x+8,y-6);line(x+8,y-6,x+3,y-1);line(x+8,y-6,x+13,y-1)
	case idThemeCycle:ellipse(x+1,y-7,x+16,y+8);line(x+8,y-7,x+8,y+8)
	case idTrayClickCycle:line(x+2,y-8,x+2,y+8);line(x+2,y-8,x+14,y+2);line(x+14,y+2,x+8,y+3);line(x+8,y+3,x+12,y+8)
	case idNotifyToggle:ellipse(x+4,y-6,x+13,y+4);line(x+2,y+4,x+15,y+4);line(x+7,y+7,x+10,y+7)
	case idExportCycle:rect(x,y-8,x+17,y+8);line(x+8,y-5,x+8,y+3);line(x+4,y,x+8,y+4);line(x+12,y,x+8,y+4)
	case idNameCycle:line(x,y-6,x+17,y-6);line(x,y,x+13,y);line(x,y+6,x+16,y+6)
	case idOCRLanguageCycle:rect(x,y-7,x+17,y+7);line(x+4,y-3,x+13,y-3);line(x+4,y+2,x+13,y+2)
	case idHistory,idHistoryToggle,idHistoryLimit,idHistoryPrev,idHistoryNext,idHistoryOpenPictures,idHistoryOpenVideos:
		if id==idHistoryPrev{line(x+13,y-7,x+4,y);line(x+4,y,x+13,y+7)}else if id==idHistoryNext{line(x+4,y-7,x+13,y);line(x+13,y,x+4,y+7)}else if id==idHistoryOpenPictures||id==idHistoryOpenVideos{line(x,y-4,x+7,y-4);line(x+7,y-4,x+9,y-7);line(x+9,y-7,x+17,y-7);rect(x,y-4,x+17,y+8)}else{ellipse(x+1,y-7,x+16,y+8);line(x+8,y-4,x+8,y);line(x+8,y,x+12,y+3)}
	case idOpen,idOpenVideos:line(x,y-4,x+7,y-4);line(x+7,y-4,x+9,y-7);line(x+9,y-7,x+17,y-7);rect(x,y-4,x+17,y+8)
	case idDiag:line(x+1,y,x+6,y+6);line(x+6,y+6,x+16,y-7)
	case idWebsite:rect(x,y-6,x+12,y+7);line(x+8,y-8,x+17,y-8);line(x+17,y-8,x+17,y+1);line(x+17,y-8,x+7,y+2)
	case idAbout:ellipse(x+1,y-8,x+16,y+8);line(x+8,y-1,x+8,y+5);line(x+8,y-5,x+8,y-5)
	default:ellipse(x+3,y-5,x+14,y+6)
	}
}
func drawCard(dc uintptr,r w.RECT,dark bool){bg:=w.RGB(255,255,255);border:=w.RGB(225,231,241);if dark{bg=w.RGB(18,24,35);border=w.RGB(38,49,68)};fillRoundedRect(dc,r,18,bg,border)}
func drawSectionLabel(dc uintptr,x,y int32,text string,dark bool){c:=w.RGB(71,84,105);if dark{c=w.RGB(147,164,192)};w.ProcSetBkMode.Call(dc,w.TRANSPARENT);w.ProcSetTextColor.Call(dc,c);w.ProcSelectObject.Call(dc,uintptr(fontRegular));w.ProcTextOutW.Call(dc,uintptr(x),uintptr(y),uintptr(unsafe.Pointer(w.UTF16(text))),uintptr(len([]rune(text))))}
