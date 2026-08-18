# Snapvera 1.0.0 Build Report

**Product:** Snapvera  
**Version:** 1.0.0  
**Publisher:** Brendigo  
**Website:** https://snapvera.com.hr/  
**Release channel:** Production baseline

## Build targets

Windows release artifacts are generated for:

- Windows x64: installed core, Setup, Portable and Uninstaller
- Windows ARM64: installed core, Setup, Portable and Uninstaller

The primary public aliases are `Snapvera-Setup.exe` (Windows x64), `Snapvera-Portable.exe` (Windows x64) and `Snapvera-Source.zip`.

## Quality gates

The release pipeline requires the following before packaging:

- `go test ./internal/...`
- source/catalog verification
- 41 complete language catalogs with identical keys
- build metadata consistency against the root `VERSION`
- stable version-independent single-instance identity
- `go vet` for internal packages and Windows command packages (the `unsafeptr` analyzer is disabled only for intentional Win32 callback/handle interop conversions)
- Windows x64 and ARM64 builds
- PE32+ architecture verification
- `DYNAMIC_BASE`, `HIGH_ENTROPY_VA` and `NX_COMPAT` verification
- exact Setup payload verification against the corresponding installed core and Uninstaller
- deterministic source/portable/icon packaging and SHA-256 manifests
- clean-room rebuild from the packaged source before final delivery

GitHub workflows additionally provide a native `windows-latest` x64 `--self-test` when the repository is pushed to GitHub. The Linux build host used to assemble the local artifacts cannot replace a real interactive Win32 GUI click-test.

## Stability and security changes in 1.0.0

- central `VERSION` + `internal/buildinfo` source of truth
- version-independent single-instance mutex so upgrades cannot start a second independent tray instance
- safer settings recovery: malformed JSON is quarantined and defaults are restored without applying partial state
- bounded Go memory target with selective large-image memory return instead of forced full GC after routine operations
- transaction-based two-file Setup payload replacement (`Snapvera.exe` + `Uninstall.exe`)
- Setup waits for a running Snapvera instance to terminate before file replacement
- Installed Apps metadata includes version, publisher, install location, icon path, install date and estimated size
- Setup treats shell/registry metadata failures as recoverable warnings after a successful payload commit, avoiding a false "files were not installed" state
- Uninstaller validates the exact expected per-user install directory and its temporary cleanup executable before registry or filesystem deletion
- Uninstaller reports failure if its cleanup process cannot be launched or Snapvera cannot be stopped
- Portable mode remains independent of Snapvera registry settings/startup registration

## Functional production baseline

Snapvera 1.0.0 includes screenshot capture, native editor, OCR adapter for separately installed Tesseract, local MJPEG AVI screen recording, export/naming presets, History, Pin to screen, tray workflow, hotkeys, multi-monitor capture and local-first storage.

The 1.0.0 recorder does **not** claim system audio, microphone recording or H.264/MP4 support. Tesseract is not bundled with Snapvera.

## Code signing

The generated Windows binaries are not Authenticode-signed because no Brendigo code-signing certificate/private signing material is available in this build environment. Publisher text/Installed Apps metadata is not a substitute for a cryptographic signature.

## Clean-room reproducibility result

During release finalization, the packaged source archive was extracted into an empty directory, its source SHA-256 manifest was verified, the source verifier and internal unit tests were run, and all Windows targets were rebuilt from that extracted source. The resulting x64/ARM64 Portable, Installed, Setup and Uninstaller executables matched the release executables **8/8 byte-for-byte by SHA-256**.

The final delivery repeats this verification after the report/repository metadata is frozen so the downloadable source archive remains the actual verified source of the delivered binaries.

## GitHub import hardening

Prije prijenosa u `bren-wp/snapvere` dodatno su uklonjene duplicirane i razvojne datoteke: tri kopije translation kataloga zamijenjene su jednim `internal/i18n/messages.json`, tri command-level ICO kopije zamijenjene su jednim `internal/resources/snapvera.ico`, uklonjen je commitani `__pycache__` i dvije jednokratne migracijske skripte.

Dodani su testovi za i18n oblik kataloga, embedded resource, unsupported History schema i duplicirane installer destination putanje. Build skripte sada imaju cleanup i formatting gate prije release builda.
