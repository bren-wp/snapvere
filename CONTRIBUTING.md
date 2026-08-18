# Contributing to Snapvera

Snapvera is a native Windows screenshot and screen-recording utility maintained by Brendigo.

## Development checks

```bash
go test ./internal/...
python3 tools/verify_release.py .
./tools/build_windows.sh
```

On Windows, use `tools/build_windows.ps1` and run the generated x64 portable executable with `--self-test` before opening a pull request.

Keep capture, editor, recorder and installer changes small enough to review independently. New user-visible text must be added consistently to all locale catalogs. Do not bundle third-party executables or libraries without a license review and an update to `THIRD_PARTY_NOTICES.md`.
