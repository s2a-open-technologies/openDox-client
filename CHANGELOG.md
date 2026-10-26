# Changelog

All notable changes to the openDox Client are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Versions are bumped via `python3 scripts/bump_version.py` (patch/minor/major) and tagged in
git (`--commit` flag), which also moves this file's `[Unreleased]` section under the new
version heading. Pushing the resulting `vX.Y.Z` tag triggers `.github/workflows/release.yml`,
which attaches that section as the GitHub Release notes; the signed installers are attached
afterwards with `scripts/release.sh`.

## [Unreleased]

## [0.8.5] — 2026-10-26

### Fixed

- **Client bleibt angemeldet** — Der Client erneuerte seine Anmeldung bisher nur beim Start. Lief er länger
  als 30 Minuten, schlugen Aufnahme, Kategorien und Status still fehl, bis man ihn neu startete. Jetzt
  erneuert er das Token vor Ablauf und bei jeder Anfrage, die es braucht. Angemeldet bleibt man, solange
  der Client mindestens einmal pro Woche läuft. Lehnt der Server die Erneuerung ab (Konto gesperrt,
  LDAP-/SSO-Gruppe entzogen), erscheint der Anmeldebildschirm. Ist der Server nur kurz nicht erreichbar,
  bleibt man angemeldet.
- **Hinweise verdecken keine Schaltflächen mehr** — Meldungen erscheinen jetzt unten rechts im Textfeld
  statt über den Schaltflächen darunter.
- **Mikrofon-Auswahl schon vor der ersten Aufnahme** — Das Mikrofon-Symbol oben war erst nach einer
  Aufnahme zu sehen, weil der Client ohne Mikrofonfreigabe nicht weiß, wie viele Mikrofone angeschlossen
  sind. Es ist jetzt von Anfang an da. Beim ersten Öffnen fragt der Client die Freigabe ab und zeigt dann
  die Liste.
- **Kein doppelter Aufnahme-Button** — Liegt ein fertiger Text vor, ist der große Aufnahme-Button
  ausgeblendet. Die neue Aufnahme startet dann der mittlere der drei Buttons darunter (oder
  **Strg+Alt+R**).

## [0.8.4] — 2026-10-23

### Added

- **Lizenztexte in allen Paketen** — MSI, Windows-Installer, Linux-Archiv und macOS-Bundle enthalten jetzt
  `LICENSE.txt` und `THIRD_PARTY_LICENSES.txt` mit den Lizenztexten aller mitgelieferten Komponenten. Im
  Einstellungen-Dialog stehen Version, Lizenz und ein Link zum Quellcode.
- **Benutzerhandbuch** — Ein Handbuch für den Client als PDF, im selben Layout wie das openDox-Handbuch
  (`docs/handbuch/`, gebaut mit `scripts/build_handbuch.sh`). Es hängt künftig an jedem Release.
- **Hell/Dunkel-Design** — Das Overlay gibt es jetzt auch hell. Umschalten über das Zahnrad in der Statusleiste
  oder das Sonnen-/Mond-Symbol auf dem Anmeldebildschirm. Standard ist weiterhin dunkel.
- **Einstellungen im Overlay** — Über das Zahnrad stellt jeder Nutzer Design, Live-Ansicht und das Verhalten der
  Mikrofontaste (Halten/Umschalten) selbst ein. Die Wahl wird in `preferences.json` im Benutzer-Konfigordner
  gespeichert und gewinnt gegenüber `settings.json` und Server-Vorgabe; „Auf Standard zurücksetzen“ stellt die
  Vorgaben wieder her. Eine frühere Wahl der Live-Ansicht wird übernommen.

### Changed

- **`settings.json` gewinnt wieder über den Server** — Bisher hat die Server-Richtlinie die lokalen Schlüssel
  praktisch immer überschrieben (der Server schickt jedes Feld mit). Jetzt liefert der Server nur die Vorgabe für
  das, was die lokale `settings.json` nicht festlegt.
- **Server-Richtlinie nur beim Start** — Der Client holt sie einmal beim Start statt alle 10 Minuten.

- **Orb wie im Web-Frontend** — Ist die Live-Ansicht aus, zeigt der Client jetzt denselben cyanfarbenen,
  weich leuchtenden Orb wie die openDox-Weboberfläche statt des roten, zackigen.
- **Titel der Aufnahmen** — Aufnahmen aus dem Client heißen jetzt „Client-Aufnahme“ statt „Overlay-Aufnahme“.

### Fixed

- **„Neue Aufnahme“ hängt an die alte Aufnahme an** — Gab es beim Programmstart eine unfertige Aufnahme
  (z. B. nach einem Verbindungsabbruch), hat der Client sie nicht nur einmal fortgesetzt, sondern bei
  jeder weiteren „Neuen Aufnahme“ wieder dieselbe Sitzung verwendet und den Text an die alte Aufnahme
  angehängt. Jetzt wird sie nur beim nächsten Start einmal fortgesetzt, danach entsteht jedes Mal eine
  neue Aufnahme.
- **Windows friert kurz nach dem Start ein / Bluescreen** — Die Suche nach Diktiermikros (beim Start und
  danach alle 5 s) hat jedes HID-Gerät (Maus, Tastatur, ...) geöffnet und abgefragt und dafür jedes Mal
  das ganze Programm als Unterprozess gestartet. Auf einer Windows-VM (ARM, VirtualBox) hat das Windows
  einfrieren lassen. Jetzt werden nur noch Geräte von Philips und Nuance geöffnet (Auswahl anhand des
  Gerätepfads), und der Unterprozess wird nur noch unter macOS verwendet.
- **Hinweis-Meldungen verdecken die Buttons** — Meldungen (Toasts) erscheinen jetzt unterhalb der
  Statusleiste über dem Textbereich statt am oberen Rand, wo sie Kategorie, Mikrofon, Logout und
  Schließen verdeckt haben.
- **settings.json bei MSI-Update** — Die MSI legt `settings.json` nur noch an, wenn sie fehlt. Eine
  vorhandene (z. B. per `copy`/`Copy-Item`/GPO abgelegte) Datei wird bei einem Update nicht mehr mit
  der Vorlage (`localhost`) überschrieben. Eine Deinstallation entfernt sie weiterhin.
- **Installationspfad in der Doku** — `DEPLOYMENT.md` und `docs/data-mapping.md` nennen jetzt den
  tatsächlichen MSI-Pfad `%ProgramFiles%\openDox\` statt `%ProgramFiles%\openDox\openDox\`.
- **macOS: „Akzeptieren und einfügen“** — Das Overlay wird beim Einfügen per Button nicht mehr ausgeblendet
  (es blieb danach unsichtbar, und der Text kam nicht an). Stattdessen wird das zuvor aktive Programm
  wieder in den Vordergrund geholt und der Text dort eingefügt; das Overlay bleibt sichtbar.
- **macOS: Absturzmeldung „openDox neu starten“** — Die Suche nach Diktiermikros fragt nur noch Geräte von
  Philips und Nuance ab. Bisher wurden alle HID-Geräte ausgelesen, was unter macOS 26 den Such-Hilfsprozess
  gelegentlich abstürzen ließ; macOS meldete das alle paar Minuten, obwohl die App weiterlief.
- **macOS-Build** — `paste_darwin.go` ließ sich mit aktuellem macOS-SDK nicht kompilieren
  (`CGEventSourceKeyState` mit falschem Argumenttyp).

### Security
- Update github.com/labstack/echo/v4 v4.13.3 -> v4.15.3
- Update vite@8.3.2 -> vite@8.3.5
- 

## [0.8.3] — 2026-10-15

### Changed

- **Rechteinhaber** — In allen Quelldateien, `REUSE.toml` und README steht jetzt **S2A Open Technologies
  gGmbH** als Rechteinhaber (Rechte übertragen); Christopher Abanilla bleibt Urheber (`AUTHORS`). Die
  Lizenz (AGPL-3.0-or-later) ist unverändert.
- **Firmenname** — `companyName` in `wails.json` ist jetzt "S2A Open Technologies gGmbH" (Herausgeber in den
  Windows-Installern, Hersteller im MSI, Dateieigenschaften der `.exe`). Der **NSIS-Installer** legt den
  Firmennamen auch in den Installationsordner und den Deinstallations-Eintrag; eine bestehende `.exe`-Installation
  wird deshalb nicht aktualisiert, sondern daneben neu installiert. Das **MSI** installiert nach
  `Program Files\openDox` und aktualisiert wie bisher.
- Kontaktadresse für Sicherheitsmeldungen, Verhaltenskodex und Autoreintrag ist jetzt
  `christopher.abanilla@s2a-ot.digital`.

- **Installer aus der CI** — Beim Tag-Push baut der Release-Workflow jetzt den Windows-Installer
  (`opendox-client-amd64-installer.exe`), das MSI (`opendox-client-installer.msi`) und ein
  Linux-Binary (`opendox-client-linux-amd64.tar.gz`) und hängt sie unter den festen Namen an das Release,
  sodass die Download-Links der Server-GUI ohne manuellen Upload funktionieren. Die CI-Builds sind
  **nicht signiert** (Hinweis in den Release-Notizen); signierte Builds ersetzen sie über
  `scripts/release.sh`. Für ein bestehendes Release: `gh workflow run release.yml -f tag=vX.Y.Z`.
  macOS bleibt lokal.

## [0.8.2] — 2026-10-15

### Fixed

- **Fortsetzen einer Aufnahme auf der falschen Server-Instanz** — Läuft der Server in
  mehreren Instanzen und landet die Wiederverbindung auf einer, die das bisherige Audio
  der Aufnahme nicht hat, lehnt der Server sie jetzt mit dem Fehlercode
  `recording_audio_unavailable` ab (statt in einer neuen, leeren Datei weiterzuschreiben).
  Der Client zeigt dazu eine eigene Meldung statt „Verbindung unterbrochen“. Ältere
  Server senden den Code nicht, dann bleibt das Verhalten wie bisher.

## [0.8.1] — 2026-10-08

### Added

- **Client-Richtlinie vom openDox-Server** (`policy_sync.go`) — Zwischenablage-Schutz,
  Einfüge-Allow-Liste, Diktiermikro-Modus und Live-Ansicht werden zentral im Server
  (Admin → Clients) gepflegt. Der Client holt sie beim Start und danach alle 10 Minuten
  (`GET /api/v1/settings/client-config/`, ohne Login), Werte vom Server haben Vorrang vor
  der lokalen `settings.json`. Der letzte Stand wird pro Server-Adresse zwischengespeichert
  und gilt auch, wenn der Server beim Start nicht erreichbar ist. Benötigt einen Server mit
  diesem Endpunkt; ein älterer Server liefert 404, dann gilt wie bisher die lokale Datei.
  Änderungen wirken zur Laufzeit (Mic-Modus pro Tastendruck, Live-Ansicht per
  `policy:changed`-Event).
- **Open-Source-Infrastruktur** — `CHANGELOG.md`, `SECURITY.md`, `CONTRIBUTING.md`,
  `CODE_OF_CONDUCT.md`, `AUTHORS`, Issue- und PR-Templates, GitHub-Actions-CI (gofmt, `go vet`,
  Tests, Frontend-Typecheck und -Build, REUSE), Dependabot (Go, npm, Actions) und ein
  Release-Workflow, der aus dem Tag und dem Changelog die GitHub-Release-Notizen erzeugt.
- **Skripte** — `scripts/bump_version.py` (Version in `wails.json` und `frontend/package.json`,
  Changelog, Commit und Tag), `scripts/check_licenses.py` (Lizenzprüfung und
  `THIRD_PARTY_NOTICES.md`), `scripts/add_spdx_headers.py`, `scripts/release.sh` (Installer als
  GitHub-Release-Dateien unter festen Namen; der Server verlinkt darauf).
- **REUSE-Konformität** — SPDX-Kurzform-Header in allen Quelldateien, `REUSE.toml`,
  `LICENSES/`; `reuse lint` ist grün.
- **Dokumentation** — `docs/opendox-client.md` (Überblick), `docs/data-mapping.md`
  (Datenfluss, Grundlage für ein Verzeichnis von Verarbeitungstätigkeiten) und
  `docs/audit-checklist-opendox-client.md` (Selbstauskunft für Audits).
- **Zweckbestimmung und Rahmenwerke** — `docs/zweckbestimmung.md` (Abgrenzung zum
  Medizinprodukt) und `docs/rahmenwerke.md` (Einordnung in ISO 27001, NIS2, CRA, MDR, DSGVO).
- **SBOM** — der Release-Workflow hängt CycloneDX-SBOMs (Go-Module, produktive npm-Pakete)
  an jedes Release an.

### Changed

- Build-Skripte (`compile.sh`, `build-macos.sh`, `build-windows-msi.sh`) liegen jetzt in
  `scripts/`; `SECURITY.md`, `CONTRIBUTING.md` und `CODE_OF_CONDUCT.md` in `.github/`.
- `DISCLAIMER.md`, Audit-Checkliste und Data-Mapping: Hold-Modus des Diktiermikros fügt ohne
  weiteren Klick ein (Ausnahme vom Bestätigungsschritt); TLS-Hinweise präzisiert (der Server
  verlangt `https://`).

- `settings.json.template` enthält nur noch `server_url`; die übrigen Schlüssel
  (`clipboard_*`, `insert_allowlist`, `dictation_mic_mode`, `live_view`) sind optionale
  Rückfallwerte. `README.md`, `DEPLOYMENT.md` und `AGENTS.md` entsprechend angepasst.
- `main.go`, `accessibility_stub.go` und `dictation_mic_protocol_test.go` mit `gofmt`
  formatiert (Voraussetzung für die CI-Prüfung).

### Removed

- Der Microsoft-WebView2-Bootstrapper (`build/windows/installer/tmp/`) liegt nicht mehr im
  Repository; Wails lädt ihn beim NSIS-Build selbst herunter.

## [0.8.0] — 2026-10-01

Erster getaggter Stand; fasst die bisherige Entwicklung seit August 2026 zusammen.

### Added

- **Aufnehmen, live mitlesen, einfügen** — Always-on-top-Overlay (Wails v2, Go + React) mit
  Live-Transkription über einen openDox-Server, Kategorien und dem expliziten Schritt
  „Akzeptieren und einfügen“.
- **Globale Hotkeys** — Aufnahme, Pause, Einfügen, Halten-zum-Sprechen und Overlay
  aus-/einblenden (`Strg+Alt+R/P/V/D/H`, macOS `Cmd+Option`).
- **Diktiermikro-Tasten** (Philips SpeechMike / Nuance PowerMic) mit Hold-Modus
  (Push-to-Talk, Standard) oder Toggle-Modus.
- **macOS-Unterstützung** — `build-macos.sh`, Einfügen per `Cmd+V` über `CGEventPost`,
  Mikrofon-Berechtigung der WebView, Prüfung der Bedienhilfen-Berechtigung.
- **SSO-Login (OIDC)** über den System-Browser mit Loopback-Redirect nach RFC 8252.
- **Windows-Installer** — MSI (`build-windows-msi.sh`, wixl) und NSIS; Server-Adresse in einer
  `settings.json` neben dem Programm, Anleitung für Intune/SCCM/GPO in `DEPLOYMENT.md`.
- **Zwischenablage-Schutz (Windows)** — Text aus Verlauf und Cloud-Zwischenablage heraushalten,
  nach 30 s leeren, wenn unverändert; Anzeige des Einfügeziels und optionale Allow-Liste.
- **Live-Ansicht abschaltbar** (`live_view`, Umschalter im Overlay; der Server startet dann keine
  Live-Transkription für die Aufnahme).
- **AGPL-3.0-or-later** mit Lizenzkopf in den Quelldateien und `DISCLAIMER.md`.

### Changed

- Server-Adresse wird zur Laufzeit aus `settings.json` gelesen (kein Rebuild pro Kunde).
- Windows: Einfügen per `Strg+V` statt Zeichen für Zeichen (in Citrix-Clients getippt); auf allen
  Plattformen erst nach dem Loslassen der Hotkey-Modifier. Das Overlay nimmt dem Zielprogramm den
  Fokus nicht mehr weg (`WS_EX_NOACTIVATE`).

### Fixed

- Verstümmelter Text bei mehrfach schnell ausgelöstem „Akzeptieren und einfügen“.
- Absturz des Diktiermikro-Codes unter macOS 26.
- Native Konsolenausgabe von GTK/WebKit im Normalbetrieb stillgelegt; `--debug` öffnet unter
  Windows eine sichtbare Konsole.
- OIDC-Callback und automatische WebView-Berechtigungen gehärtet.

[Unreleased]: https://github.com/daemolition/openDox-client/compare/v0.8.5...HEAD
[0.8.5]: https://github.com/daemolition/openDox-client/compare/v0.8.4...v0.8.5
[0.8.4]: https://github.com/daemolition/openDox-client/compare/v0.8.3...v0.8.4
[0.8.3]: https://github.com/daemolition/openDox-client/compare/v0.8.2...v0.8.3
[0.8.2]: https://github.com/daemolition/openDox-client/compare/v0.8.1...v0.8.2
[0.8.1]: https://github.com/daemolition/openDox-client/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/daemolition/openDox-client/releases/tag/v0.8.0
