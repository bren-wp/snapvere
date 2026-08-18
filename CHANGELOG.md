# Snapvera Changelog

## 1.0.0 — Production Baseline

Dodatni GitHub hardening prije javnog importa:

- centraliziran jedan kanonski i18n katalog za aplikaciju, Setup i Uninstaller
- centralizirana jedna kanonska aplikacijska ICO datoteka
- uklonjen commitani Python `__pycache__` i jednokratne migracijske skripte
- sigurniji fallback za settings, log, history i output direktorije
- History odbija nepoznatu schema verziju umjesto tihog učitavanja
- installer transakcija odbija duplicirane destination putanje
- recorder stop je responzivniji tijekom frame-timinga
- build skripte uvijek čiste privremene Setup payload datoteke, čak i kod greške
- Bash Windows build sada vet-a i Windows-specifične internal pakete s odgovarajućim GOOS/GOARCH

- centralizirana verzija i branding kroz `VERSION` + `internal/buildinfo`
- stabilan single-instance mutex bez stare 0.x verzije u identitetu
- sigurnije učitavanje i recovery oštećenih postavki
- ne-nulti panic exit code i ranija inicijalizacija dijagnostičkog loga
- moderniziran Setup s informacijama o verziji/arhitekturi, Start with Windows opcijom i boljim Installed apps metapodacima
- installer više ne koristi temp fallback ako `LOCALAPPDATA` nije dostupan i ne nastavlja update ako se stara instanca ne ugasi
- jedinstveni backup nazivi u installer transakciji
- GitHub CI/Release workflowi i user-friendly release aliasi
- source/release dokumentacija podignuta na 1.0.0

## 0.13.0 — UI & Stability Refinement

### Dizajn i UX
- odvojene **Snimka zaslona** i **Video** kartice u Postavkama; više nisu jedna nejasna grupa
- screenshot akcije koriste plavo-ljubičasti akcent, video akcije zaseban crveni akcent
- novi dvoredni button renderer: kategorija/vrijednost + glavni tekst ostaju čitljivi i kod duljih prijevoda
- veći padding, ikon plate, radius, focus/pressed/hot stanja i kvalitetniji kontrast gumba
- editor toolbar reorganiziran iz dva zbijena reda u tri čitljiva reda
- modernizirani Setup i Uninstall prozori i gumbi
- poboljšane native video ikonice: Full, Region i Stop sada su međusobno vizualno različite
- UI icon pack proširen na 36 SVG ikonica, uključujući zasebne Screenshot i Video ikone

### Imenovanje i izvoz
- screenshot i video datoteke uvijek imaju različite tipove u nazivu: `Snapvera-Screenshot-...` i `Snapvera-Video-...`
- Compact preset koristi `SV-IMG-...` i `SV-VID-...`
- Technical preset uključuje tip medija i način snimanja
- dodane milisekunde u vremenske presetove
- dodana provjera postojećih izlaznih datoteka kako restart aplikacije ne bi mogao prepisati snimku s istim nazivom
- naming logika izdvojena u testabilni `internal/naming` modul

### Stabilnost i performanse
- uklonjeno prisilno `GC + FreeOSMemory` nakon svake male snimke; agresivno vraćanje memorije sada se koristi samo nakon velikih capture površina
- završetak recorder sesije više ne dira Settings kontrole iz pozadinske gorutine; UI refresh se marshalira na tray/UI thread
- zadržan jedan reusable DIB capture surface i jedan reusable JPEG buffer po video sesiji
- History i Pin ostaju local-first i ograničeni retention/pixel budget pravilima

### Installer i sigurnost
- Setup koristi transakcijsku zamjenu `Snapvera.exe + Uninstall.exe` s prethodnim stagingom i rollbackom
- transakcijska logika izdvojena u `internal/installtx` i pokrivena unit testovima
- Uninstaller više ne prihvaća bilo koju putanju koja samo sadrži riječ `snapvera`; cleanup je ograničen na `%LOCALAPPDATA%\\Programs\\Snapvera`
- privremeni uninstaller sada se stvara sigurnim `CreateTemp` postupkom umjesto predvidljivog PID naziva
- spremanje postavki koristi jedinstvenu temp datoteku, `Sync` i Windows `MoveFileEx(...REPLACE_EXISTING|WRITE_THROUGH)`

### Quality gates
- unit testovi: AVI, History, Naming i Install Transaction
- Windows x64 i ARM64 build
- PE verifier za Portable / Installed / Setup / Uninstall
- 41 jezik × 112 UI stringova
- 36 SVG UI ikonica