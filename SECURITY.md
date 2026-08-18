# Snapvera Security — 1.0.0

Snapvera je local-first desktop aplikacija. Screenshotovi, videozapisi, history podaci i OCR privremene datoteke ne šalju se automatski na Snapvera/Brendigo servere.

## 1.0.0 sigurnosne mjere

- Windows PE release provjera zahtijeva PE32+, odgovarajuću x64/ARM64 arhitekturu, `DYNAMIC_BASE`, `HIGH_ENTROPY_VA` i `NX_COMPAT`.
- Portable i Installed buildovi ostaju odvojeni; Setup mora sadržavati Installed core, ne Portable build.
- Installer priprema sve payload datoteke prije zamjene i koristi rollback transakciju s jedinstvenim backup nazivima.
- Uninstaller rekurzivno briše samo očekivani `%LOCALAPPDATA%\\Programs\\Snapvera` direktorij.
- Settings se zapisuju preko jedinstvene temp datoteke, `Sync` i atomskog Windows replacea.
- Oštećene settings datoteke ne primjenjuju se djelomično.
- OCR pokreće točno pronađenu lokalnu executable putanju bez `cmd.exe`, PowerShella ili shell stringa i ima timeout.
- OCR privremena slika briše se nakon obrade.
- Capture, recording i Pin imaju zaštitne memory/pixel limite.
- Single-instance identitet je stabilan između verzija kako bi se izbjegli dvostruki tray/hotkey procesi tijekom nadogradnje.

## Privatnost

Snapvera 1.0.0 nema ugrađenu telemetriju, oglasni SDK ili automatski cloud upload.

## Code signing

Lokalno generirani 1.0.0 EXE artefakti nisu Authenticode potpisani jer izdavački certifikat nije dostavljen. PE metadata `Publisher: Brendigo` nije isto što i kriptografski code signing. Za javnu distribuciju preporučuje se EV/OV Authenticode certifikat i potpisivanje finalnih release artefakata.

## Prijava ranjivosti

Sigurnosni problem prijaviti privatnim kanalom Brendigu. U javnom issueju ne objavljivati exploit detalje, lozinke, privatne screenshotove, tokene ili druge osjetljive podatke.
