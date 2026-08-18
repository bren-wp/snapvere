#!/usr/bin/env python3
from pathlib import Path
import struct,sys,hashlib
root=Path(sys.argv[1])
version=(Path(__file__).resolve().parents[1]/'VERSION').read_text(encoding='utf-8').strip()
expected={
 'x64':0x8664,
 'arm64':0xAA64,
}
errors=[]
for arch,machine_expected in expected.items():
    files={k:root/f'Snapvera-{version}-windows-{arch}{suffix}' for k,suffix in {
      'portable':'-portable.exe','installed':'.exe','setup':'-setup.exe','uninstall':'-uninstall.exe'}.items()}
    for kind,p in files.items():
        if not p.exists(): errors.append(f'missing {p.name}'); continue
        b=p.read_bytes()
        if b[:2]!=b'MZ': errors.append(f'{p.name}: missing MZ'); continue
        pe=struct.unpack_from('<I',b,0x3c)[0]
        if b[pe:pe+4]!=b'PE\0\0':errors.append(f'{p.name}: missing PE');continue
        machine=struct.unpack_from('<H',b,pe+4)[0]
        magic=struct.unpack_from('<H',b,pe+24)[0]
        dllchars=struct.unpack_from('<H',b,pe+24+70)[0]
        if machine!=machine_expected:errors.append(f'{p.name}: wrong machine {machine:#x}')
        if magic!=0x20b:errors.append(f'{p.name}: not PE32+')
        for bit,name in [(0x20,'HIGH_ENTROPY_VA'),(0x40,'DYNAMIC_BASE'),(0x100,'NX_COMPAT')]:
            if not dllchars&bit:errors.append(f'{p.name}: missing {name}')
        low=b.lower()
        for marker in (b'vcruntime',b'msvcp140',b'api-ms-win-crt'):
            if marker in low:errors.append(f'{p.name}: unexpected runtime marker {marker!r}')
        for marker in (b'development build', b'debug build', b'developer mode'):
            if marker in low: errors.append(f'{p.name}: forbidden production marker {marker!r}')
        for stale in (b'0.7.0',b'0.8.0',b'0.9.0',b'0.10.0',b'0.11.0',b'0.12.0'):
            if stale in b: errors.append(f'{p.name}: stale version text {stale!r}')
        if kind != 'uninstall':
            for required in (version.encode(), b'Brendigo', b'snapvera.com.hr'):
                if required not in b: errors.append(f'{p.name}: missing release metadata {required!r}')
    if all(p.exists() for p in files.values()):
        setup=files['setup'].read_bytes()
        if files['installed'].read_bytes() not in setup:errors.append(f'{arch} setup missing installed payload')
        if files['uninstall'].read_bytes() not in setup:errors.append(f'{arch} setup missing uninstall payload')
        if files['portable'].read_bytes() in setup:errors.append(f'{arch} setup incorrectly embeds portable payload')
if errors:
    print('\n'.join('ERROR: '+e for e in errors));sys.exit(1)
print('Windows artifact verification: OK')
