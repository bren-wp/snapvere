#!/usr/bin/env python3
from pathlib import Path
import hashlib, zipfile, os, shutil, sys

root=Path(__file__).resolve().parents[1]
dist=root/'dist'
version=(root/'VERSION').read_text(encoding='utf-8').strip()
dist.mkdir(exist_ok=True)
fixed=(2026,8,18,21,0,0)

def source_files():
    out=[]
    for p in root.rglob('*'):
        if not p.is_file(): continue
        rel=p.relative_to(root)
        if rel.parts[0] in {'dist','.git'}: continue
        if '__pycache__' in rel.parts: continue
        if p.suffix.lower() in {'.exe','.avi'}: continue
        out.append((rel,p))
    return sorted(out,key=lambda x:x[0].as_posix())

def write_zip(path, entries):
    if path.exists(): path.unlink()
    with zipfile.ZipFile(path,'w',compression=zipfile.ZIP_DEFLATED,compresslevel=9) as z:
        for arc,src in entries:
            data=Path(src).read_bytes()
            zi=zipfile.ZipInfo(str(arc).replace('\\','/'),fixed)
            zi.compress_type=zipfile.ZIP_DEFLATED
            arc_path=Path(str(arc))
            executable = arc_path.parts and arc_path.parts[0] == 'tools' and arc_path.suffix.lower() in {'.sh','.py'}
            mode = 0o100755 if executable else 0o100644
            zi.external_attr=(mode & 0xffff)<<16
            z.writestr(zi,data)

def make_source():
    entries=source_files()
    write_zip(dist/f'Snapvera-{version}-source.zip',entries)
    manifest=[]
    for rel,p in entries:
        manifest.append(f'{hashlib.sha256(p.read_bytes()).hexdigest()}  {rel.as_posix()}')
    (dist/'SOURCE-MANIFEST-SHA256.txt').write_text('\n'.join(manifest)+'\n',encoding='utf-8')

def make_portables():
    for arch in ('x64','arm64'):
        exe=dist/f'Snapvera-{version}-windows-{arch}-portable.exe'
        entries=[(exe.name,exe),(Path('PORTABLE_README.txt'),root/'PORTABLE_README.txt'),(Path('SnapveraTools/tesseract/README.txt'),root/'SnapveraTools/tesseract/README.txt')]
        write_zip(dist/f'Snapvera-{version}-windows-{arch}-portable.zip',entries)

def make_icons():
    entries=[]
    for p in [root/'internal/resources/snapvera.ico',*sorted((root/'assets/icons').glob('*')), *sorted((root/'assets/ui-icons').glob('*'))]:
        if not p.is_file():
            continue
        arc = Path('snapvera.ico') if p.name == 'snapvera.ico' else p.relative_to(root/'assets')
        entries.append((arc,p))
    write_zip(dist/f'Snapvera-{version}-icons.zip',entries)

def make_friendly_aliases():
    aliases = {
        f'Snapvera-{version}-windows-x64-setup.exe': 'Snapvera-Setup.exe',
        f'Snapvera-{version}-windows-x64-portable.exe': 'Snapvera-Portable.exe',
        f'Snapvera-{version}-source.zip': 'Snapvera-Source.zip',
    }
    for src_name, alias_name in aliases.items():
        src=dist/src_name
        if src.exists(): shutil.copy2(src, dist/alias_name)

def copy_docs():
    for name in ['README.md','RELEASE_NOTES.md','BUILD_REPORT.md','SECURITY.md','THIRD_PARTY_NOTICES.md','CHANGELOG.md','LICENSE']:
        shutil.copy2(root/name,dist/name)

def sums():
    files=[p for p in sorted(dist.iterdir()) if p.is_file() and p.name not in {'SHA256SUMS.txt',f'Snapvera-{version}-DELIVERY.zip'}]
    lines=[f'{hashlib.sha256(p.read_bytes()).hexdigest()}  {p.name}' for p in files]
    (dist/'SHA256SUMS.txt').write_text('\n'.join(lines)+'\n',encoding='utf-8')

def delivery():
    files=[p for p in sorted(dist.iterdir()) if p.is_file() and p.name!=f'Snapvera-{version}-DELIVERY.zip']
    write_zip(dist/f'Snapvera-{version}-DELIVERY.zip',[(p.name,p) for p in files])

cmd=sys.argv[1] if len(sys.argv)>1 else 'all'
if cmd in ('source','all'): make_source()
if cmd=='source': sys.exit(0)
if cmd in ('artifacts','all'):
    make_portables(); make_icons(); copy_docs(); make_source(); make_friendly_aliases(); sums(); delivery()
