# Publishing Snapvera to GitHub

Target repository: `bren-wp/snapvere` (or an organization repository if Brendigo chooses that owner).

The source tree already contains GitHub Actions workflows:

- `.github/workflows/ci.yml` — source validation, unit tests, Windows build and a native Windows x64 `--self-test`.
- `.github/workflows/release.yml` — when a tag such as `v1.0.0` is pushed, builds the release on `windows-latest`, runs the native self-test, packages artifacts, verifies SHA-256 and creates the GitHub Release.

Primary GitHub Release assets are:

- `Snapvera-Setup.exe`
- `Snapvera-Portable.exe`
- `Snapvera-Source.zip`
- versioned x64/ARM64 setup and portable packages
- `SHA256SUMS.txt`

The locally generated `.exe` files are unsigned unless an Authenticode certificate is supplied. Do not describe unsigned binaries as code-signed.
