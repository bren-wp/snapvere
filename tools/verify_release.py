#!/usr/bin/env python3
from pathlib import Path
import json, re, sys

root=Path(sys.argv[1]).resolve()
version=(root/'VERSION').read_text(encoding='utf-8').strip()
errors=[]

if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+", version):
    errors.append(f'VERSION is not semantic x.y.z: {version!r}')

msg_files=[root/'internal/i18n/messages.json']
parsed=[]
for p in msg_files:
    try: d=json.loads(p.read_text(encoding='utf-8'))
    except Exception as e:
        errors.append(f'{p}: invalid json: {e}'); continue
    parsed.append((p,d))
    langs=d.get('languages',{})
    if len(langs)<41: errors.append(f'{p}: expected >=41 languages, got {len(langs)}')
    en=langs.get('en',{})
    expected=set(en)
    if len(expected)<112: errors.append(f'{p}: too few translation keys: {len(expected)}')
    if set(d.get('keys',[]))!=expected: errors.append(f'{p}: metadata keys do not match English catalog')
    for code,items in langs.items():
        if set(items)!=expected:
            missing=sorted(expected-set(items)); extra=sorted(set(items)-expected)
            errors.append(f'{p}: {code} key mismatch missing={missing[:4]} extra={extra[:4]}')
        blanks=[k for k,v in items.items() if not isinstance(v,str) or not v.strip()]
        if blanks: errors.append(f'{p}: {code} blank translations: {blanks[:4]}')
for size in [16,20,24,32,40,48,64,96,128,256]:
    p=root/f'assets/icons/snapvera-{size}.png'
    if not p.exists() or p.stat().st_size<100: errors.append(f'missing/invalid icon {p.name}')
if (root/'assets/snapvera.ico').exists(): errors.append('duplicate assets/snapvera.ico must not exist; use internal/resources/snapvera.ico')
if not (root/'internal/resources/snapvera.ico').exists(): errors.append('missing canonical internal/resources/snapvera.ico')
ui=list((root/'assets/ui-icons').glob('*.svg'))
if len(ui)<36: errors.append(f'expected >=36 UI SVG icons, got {len(ui)}')
if list(root.rglob('tesseract.exe')): errors.append('Tesseract executable must not be bundled in this release')
for required in [root/'cmd/snapvera/history_windows.go', root/'cmd/snapvera/pin_windows.go', root/'internal/history/store.go', root/'internal/history/store_test.go', root/'internal/naming/naming.go', root/'internal/naming/naming_test.go', root/'assets/ui-icons/history.svg', root/'assets/ui-icons/pin.svg', root/'assets/ui-icons/screenshot.svg', root/'assets/ui-icons/video.svg']:
    if not required.exists(): errors.append(f'missing production workflow file: {required.relative_to(root)}')

for name in ['README.md','RELEASE_NOTES.md','SECURITY.md','PORTABLE_README.txt','BUILD_REPORT.md']:
    p=root/name
    text=p.read_text(encoding='utf-8',errors='replace') if p.exists() else ''
    if version not in text: errors.append(f'{name}: missing {version}')
    if 'Brendigo' not in text and name in ('README.md','PORTABLE_README.txt','BUILD_REPORT.md'): errors.append(f'{name}: missing Brendigo branding')
for stale in ['0.8.0','0.9.0','0.10.0','0.11.0','0.12.0','0.13.0']:
    for name in ['README.md','RELEASE_NOTES.md','SECURITY.md','PORTABLE_README.txt','BUILD_REPORT.md']:
        p=root/name
        if p.exists() and stale in p.read_text(encoding='utf-8',errors='replace'):
            errors.append(f'{name}: stale version {stale}')
for junk in ['test-recording.avi','snapvera.exe','uninstall.exe']:
    if (root/junk).exists(): errors.append(f'junk build artifact in source root: {junk}')

bi=root/'internal/buildinfo/buildinfo.go'
if not bi.exists():
    errors.append('missing internal/buildinfo/buildinfo.go')
else:
    bt=bi.read_text(encoding='utf-8',errors='replace')
    if f'Version   = "{version}"' not in bt: errors.append('buildinfo version does not match VERSION')
if 'SingleInstance.v0' in (root/'cmd/snapvera/main_windows.go').read_text(encoding='utf-8',errors='replace'):
    errors.append('single-instance identity must not be version-specific')

workflow_requirements = {
    root/'.github/workflows/ci.yml': ['windows-latest', '--self-test', 'verify_release.py'],
    root/'.github/workflows/release.yml': ['windows-latest', '--self-test', 'gh release create', 'Snapvera-Setup.exe', 'Snapvera-Portable.exe', 'Snapvera-Source.zip'],
}
for wp, needles in workflow_requirements.items():
    if not wp.exists():
        errors.append(f'missing GitHub workflow: {wp.relative_to(root)}')
        continue
    wt=wp.read_text(encoding='utf-8',errors='replace')
    for needle in needles:
        if needle not in wt:
            errors.append(f'{wp.relative_to(root)}: missing required release token {needle!r}')

for required in [root/'.gitignore', root/'CONTRIBUTING.md', root/'GITHUB_PUBLISHING.md', root/'internal/buildinfo/buildinfo_test.go', root/'internal/i18n/catalog.go', root/'internal/resources/resources.go']:
    if not required.exists():
        errors.append(f'missing production repository file: {required.relative_to(root)}')

if errors:
    for e in errors: print('ERROR:',e)
    sys.exit(1)
print(f'Release source verification: OK ({len(parsed[0][1]["languages"])} languages, {len(parsed[0][1]["languages"]["en"])} keys, {len(ui)} UI SVG icons)')
