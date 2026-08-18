# Snapvera 1.0.0 — Production Baseline

Snapvera 1.0.0 je prva 1.x baza. Fokus ovog izdanja je konzistentna verzija, stabilniji lifecycle, sigurniji installer i release proces koji je spreman za GitHub.

## Glavne promjene u 1.0.0

- jedinstveni `VERSION` + `internal/buildinfo` izvor za aplikaciju, Setup, Uninstaller i release alate
- stabilni single-instance application identity bez broja stare 0.x verzije
- sigurnije učitavanje postavki: oštećeni JSON više ne ostavlja djelomično primijenjeno stanje; vraćaju se validne zadane postavke i oštećena datoteka se čuva za dijagnostiku
- panic path vraća ne-nulti exit code umjesto tihog uspješnog izlaza
- dijagnostički log se inicijalizira prije učitavanja korisničkih postavki
- soft memory target povećan je radi stabilnosti velikih 4K edit operacija; prisilni full-GC uklonjen je iz malih OCR/recording workflowa
- recorder i dalje koristi reusable surface/buffer, uz eksplicitno vraćanje velikih buffera tek nakon velikih površina
- Setup više ne pada natrag na privremeni direktorij ako `LOCALAPPDATA` nije dostupan
- Setup prekida nadogradnju ako se postojeća Snapvera ne može uredno ugasiti
- modernizirani Setup prikazuje verziju, arhitekturu, per-user/no-admin model, instalacijsku lokaciju i Start with Windows opciju
- Windows uninstall metadata uključuje DisplayIcon, InstallDate, EstimatedSize i službene URL podatke
- installer transakcija koristi jedinstvene backup datoteke umjesto zajedničkog `.old` naziva
- GitHub CI i Release workflowi dodani u source
- release generira jednostavne primarne nazive `Snapvera-Setup.exe`, `Snapvera-Portable.exe` i `Snapvera-Source.zip` uz versionirane x64/ARM64 pakete

## Funkcije uključene u 1.0.0

- screenshot: region, active window, full virtual desktop, delay, multi-monitor
- tray + global hotkeys s fallback kombinacijama
- editor: arrow, rectangle, ellipse, line, pen, marker, text, pixelate, blur, redact, crop, undo/redo, copy/save/save-as/retake/discard
- Pin to screen
- lokalna History evidencija
- local-first Tesseract OCR adapter
- lokalni MJPEG AVI video recorder
- PNG i JPEG export presetovi
- jasno odvojeni Screenshot/Video naming presetovi
- 41 jezik
- x64 i ARM64 Windows artefakti

## Poznata ograničenja

- trenutni video backend nema system audio, mikrofon ni H.264/MP4
- OCR zahtijeva zasebno instaliran Tesseract engine
- lokalni build host nije Windows, pa interaktivni Win32 GUI klik-test nije moguće iskreno tvrditi; GitHub workflow je pripremljen da na stvarnom `windows-latest` runneru izvrši x64 `--self-test`
- lokalno generirani EXE-ovi nisu Authenticode potpisani jer certifikat nije dostavljen
