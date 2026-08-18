# Snapvera 1.0.0

Snapvera je native Windows aplikacija za snimke zaslona, uređivanje, lokalno snimanje zaslona u video, OCR, povijest i local-first workflow.

**Autor / izdavač:** Brendigo  
**Službena stranica:** snapvera.com.hr  
**Verzija:** 1.0.0

## Zašto Snapvera

- tray-first rad: normalno pokretanje ostavlja Snapveru aktivnom uz Windows sat
- snimanje područja, aktivnog prozora, cijelog virtualnog desktopa i odgođeno snimanje
- editor se otvara prije spremanja; screenshot se ne sprema automatski
- strelice, oblici, olovka, marker, tekst, pixelate, blur, redact, crop, undo/redo
- Copy, Save, Save As, Retake, Discard i Pin to screen
- lokalni History bez dupliciranja screenshot/video datoteka
- lokalni MJPEG AVI screen recorder za cijeli desktop ili područje
- lokalni OCR adapter za Tesseract OCR
- PNG/JPEG export presetovi i jasno odvojeni Screenshot/Video naming presetovi
- 41 jezik
- x64 i ARM64 Windows buildovi
- Portable i per-user Setup izdanje
- nema Electron/WebView runtimea, obveznog clouda ni ugrađene telemetrije

## Screenshot workflow

1. Snapvera radi u trayu.
2. Pokreni capture preko Print Screen/fallback hotkeya ili tray izbornika.
3. Označi područje ili snimi prozor/cijeli desktop.
4. Uredi snimku u editoru.
5. Kopiraj, spremi, prikvači, ponovi snimanje ili odbaci.

Datoteke slike i videa namjerno imaju različite nazive, primjerice:

- `Snapvera-Screenshot-2026-08-18_23-40-12.123-1.png`
- `Snapvera-Video-2026-08-18_23-40-12.123-2.avi`

## Video

Recorder podržava cijeli desktop i označeno područje. Presetovi su Compact, Balanced i Smooth. Trenutni 1.0.0 backend koristi lokalni MJPEG AVI bez vanjskog video enginea. Frameovi se zapisuju izravno na disk i ne akumuliraju se svi u RAM-u.

**Napomena:** 1.0.0 ne oglašava system audio, mikrofon ili H.264/MP4 kao implementirane mogućnosti.

## OCR

OCR koristi lokalno instalirani Tesseract. Snapvera ga traži uz aplikaciju, u `SnapveraTools\\tesseract`, standardnoj `Program Files\\Tesseract-OCR` lokaciji i u PATH-u. OCR nije obvezan za osnovne capture/editor funkcije.

## Portable

`Snapvera-Portable.exe` i versionirani x64/ARM64 portable buildovi ne zahtijevaju instalaciju. Portable postavke, history indeks i dijagnostički logovi ostaju u `SnapveraData` uz EXE. Portable izdanje ne koristi Snapvera registry postavke niti Windows Startup registraciju.

## Setup

`Snapvera-Setup.exe` je primarni x64 per-user installer. Instalira u:

`%LOCALAPPDATA%\\Programs\\Snapvera`

Ne zahtijeva administratorska prava. Setup koristi transakcijsku zamjenu `Snapvera.exe + Uninstall.exe`, registrira uredne Windows Installed apps metapodatke, može uključiti Start with Windows i nakon uspjeha pokreće Snapveru u tray.

## GitHub

Službeni izvorni repozitorij za ovaj build: `https://github.com/bren-wp/snapvere`.

## Privatnost i sigurnost

- nema automatskog uploada screenshotova ili videa
- nema obveznog korisničkog računa
- nema ugrađene telemetrije ili oglasa
- OCR se izvodi lokalno kada je Tesseract dostupan
- konfiguracija se zapisuje atomskim temp-file + replace postupkom
- uninstall cleanup je ograničen na očekivani per-user instalacijski direktorij
- Windows PE release verifier provjerava PE32+, arhitekturu, ASLR, High Entropy VA i NX compatibility

## Build

Za unit testove:

```bash
go test ./internal/...
python3 tools/verify_release.py .
```

Linux/macOS build host može cross-buildati Windows artefakte:

```bash
./tools/build_windows.sh
python3 tools/package_release.py artifacts
```

Na Windowsu:

```powershell
./tools/build_windows.ps1
./dist/Snapvera-1.0.0-windows-x64-portable.exe --self-test
```

GitHub workflowi u `.github/workflows/` rade native `windows-latest` x64 self-test i automatski stvaraju Release kada se push-a tag koji odgovara `VERSION`, npr. `v1.0.0`.

## Važno o validaciji

Lokalni release iz ovog paketa buildan je na Linux hostu. Unit testovi, cross-build, PE verifikacija, payload provjera, SHA-256 i clean-room reproducibility mogu se potvrditi lokalno. Interaktivni klik-test Win32 GUI-ja zahtijeva stvarni Windows runtime; zato je u repozitorij uključen Windows-native GitHub Actions self-test i ne tvrdimo da je GUI fizički klik-testiran na Linux build hostu.
