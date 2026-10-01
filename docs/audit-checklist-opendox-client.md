# Audit-Checkliste: openDox Client (Selbstauskunft)

> Simuliert, wie ein Auditor den Client tatsächlich abklopfen würde. Jede Zeile ist mit einer
> konkreten Fundstelle belegt — keine Vermutungen. Status: ✅ implementiert/vorhanden ·
> ⚠️ teilweise/mit Einschränkung · ❌ nicht implementiert/nicht vorhanden · ➖ liegt beim Server.
> Stand: 2026-10-01 (Version 0.8.0 + Änderungen aus `[Unreleased]`); vor Wiederverwendung gegen
> den dann aktuellen Code gegenprüfen. Der Client ist nur ein Teil des Gesamtsystems — die
> serverseitigen Punkte (Verschlüsselung, Rollen, Aufbewahrung, Betroffenenrechte) stehen in
> `docs/audit-checklist-opendox.md` des openDox-Server-Repositorys. Keine Rechtsberatung.

## Security

| Prüfpunkt | Status | Fundstelle/Beleg | Kommentar |
|---|---|---|---|
| Anmeldung | ➖/✅ | Passwort-Login und SSO werden vom Server geprüft; der Client speichert kein Passwort (`AuthContext.tsx` `login`, State nur bis zum Absenden) | Rollen, Passwort-Hashing, Rate-Limiting: Server |
| Native SSO nach RFC 8252 | ✅ | `oidc.go`: Loopback-Listener auf `127.0.0.1`, Zufallsport, zufälliger Callback-Pfad (`randomToken`), 5 Min. Timeout, Einzelversuch (`oidcLoginInFlight`) | Der Server muss `native_redirect` auf eine Loopback-Adresse beschränken (Server-Seite) |
| Token-Speicherung | ⚠️ | Access-/Refresh-Token in `localStorage` des WebView-Profils (`AuthContext.tsx` `applyTokens`), nicht im OS-Schlüsselbund | Schutz nur über Benutzerprofil-Rechte + Festplattenverschlüsselung; Abmelden löscht sie |
| Tokens in URLs | ⚠️ | WebSocket-Token (`GetWSURL` in `app.go`), Refresh-Token (`/auth/refresh?refresh_token=`, `AuthContext.tsx`), SSO-Callback (`oidc.go`) | Können in Server-/Proxy-Zugriffsprotokollen landen; Query-Logging dort einschränken. Ablösung bräuchte Serveränderungen |
| Verschlüsselung in Transit erzwungen | ❌ | `settings.json` `server_url`, `deriveWSURL` in `app.go`: `http://` → `ws://` ohne Warnung; Default `http://localhost:10000` | Betreiberverantwortung — `https://` verwenden |
| TLS-Zertifikatsprüfung | ✅ | Keine `InsecureSkipVerify`/eigene `TLSClientConfig` im Code (`grep`), Go-Standardverhalten mit System-Truststore | Kein Pinning |
| Client-Richtlinie vom Server integritätsgeschützt | ⚠️ | `internal/clientpolicy/remote.go`: Abruf ohne Login; Schutz nur durch TLS | Bei `http://` manipulierbar; Cache an Server-URL gebunden (`LoadCache`) |
| Zwischenablage nach dem Einfügen geschützt | ✅ (Windows) / ❌ (sonst) | `clipboard_windows.go` → `internal/winclip` (Verlauf-/Cloud-Ausschluss, fail closed), `internal/clientpolicy/cleaner.go` (Leeren nur bei unveränderter Sequenznummer) | macOS/Linux: `clipboard_other.go` ohne Markierung und ohne Auto-Leeren |
| Einfügeziel-Kontrolle | ✅ (optional, Windows) | `insert_target*.go`, `internal/clientpolicy/policy.go` (`Allows`), Rückfrage in `acceptAndInsert` | Standardmäßig aus — der Client ist ein allgemeines Diktierwerkzeug |
| Explizite Bestätigung vor dem Einfügen | ✅ | `acceptAndInsert` in `Recorder.tsx`; Text kopiert sich nicht automatisch (`AGENTS.md` „Clipboard / accept-and-insert“) | Im Hold-Modus des Diktiermikros bewusst automatisch beim Loslassen, mit Allow-Listen-Prüfung |
| Keine sensiblen Daten in Logs | ✅ | `log.Printf`-Stellen in `*.go`: nur Pfade, Programmname, Textlänge; Fenstertitel nie geloggt (`insert_target.go`); Konsole im Normalbetrieb stillgelegt (`console_silence_*.go`) | `--debug` schaltet Ausgaben frei; `ODX_DICTATION_MIC_DEBUG=1` loggt rohe HID-Reports (nur Tasten) |
| Keine Telemetrie / Phone-Home | ✅ | Einzige ausgehende Ziele: `a.opendoxURL` (`app.go`, `policy_sync.go`, `oidc.go`), keine `http(s)://`-Literale im Frontend-Quelltext; Schriften eingebettet | Laufzeit (WebView2/WKWebView/WebKitGTK) und WebView2-Bootstrapper liegen außerhalb |
| Mikrofon-Zugriff begrenzt | ✅ | `getUserMedia` nur beim Start einer Aufnahme; Tracks werden beim Stopp beendet (`Recorder.tsx` `track.stop()`) | Linux: nur Mikrofon/Kamera automatisch erlaubt, alle anderen WebKit-Berechtigungen verweigert (`webview_perms.go`); macOS: Medienzugriff wird für die App-eigene Oberfläche erlaubt (`webview_perms_darwin.go`) |
| Globale Eingabe | ✅ | Nur registrierte Hotkeys (`golang.design/x/hotkey`), kein Tastatur-Mitschnitt; HID nur für bekannte VID/PID (`dictation_mic_protocol.go`) | Mic-Tastenbelegung noch nicht an echter Hardware verifiziert (`.agents/AGENTS.md` „Dictation mic buttons“, `docs/superpowers/specs/2026-08-06-dictation-mic-design.md`) |
| Fremdprozess-Aufrufe | ✅ | `notify-send` (Linux), `osascript` (macOS), eigener Scan-Helper (`dictation_mic_scan_helper.go`) | Feste Argumente, keine Nutzereingaben in Kommandozeilen |
| Content-Security-Policy | ❌ | `frontend/index.html`: keine CSP | Oberfläche lädt nur eingebettete Dateien (Wails-Asset-Server) |
| Code-Signierung / Integrität der Builds | ❌ | Windows: nicht signiert (Smart App Control blockiert); macOS: `build-macos.sh` signiert mit lokalem Apple-Development-Zertifikat, keine Notarisierung | Vor Kunden-Rollout nötig; Prüfsummen/Signaturen der Release-Dateien noch nicht vorhanden |
| Update-Mechanismus | ➖ (bewusst keiner) | kein Code für Updates | Verteilung per Intune/SCCM/GPO, siehe `DEPLOYMENT.md`; verhindert unbeaufsichtigten Code-Nachzug, verlangt aber aktives Patchen |
| Secrets nie im Repo | ✅ | `git grep` auf Schlüssel-/Passwort-Zuweisungen ohne Treffer; `settings.json` enthält nur die Server-Adresse | Keine Zugangsdaten im Client konfigurierbar |
| Security-relevante Tests | ⚠️ | `internal/clientpolicy/*_test.go` (Allow-Liste, Cleaner, Merge/Cache), `settings_test.go`, `hold_tracker_test.go`, `dictation_mic_protocol_test.go` (28 Tests) | Kein Test für OIDC-Loopback und Einfügen; Windows-Pakete nur per `GOOS=windows go vet` geprüft |
| Vulnerability-Disclosure-Prozess | ✅ | [`SECURITY.md`](../SECURITY.md) — E-Mail-Kontakt, 48-h-Antwort, 90-Tage-Policy | Dedizierte `security@`-Adresse offen |

## Datenschutz / DSGVO

| Prüfpunkt | Status | Fundstelle/Beleg | Kommentar |
|---|---|---|---|
| Datenflussdiagramm / ROPA-Ausgangspunkt | ✅ | [`data-mapping.md`](data-mapping.md) inkl. Mermaid-Flowchart | Ergänzt das Server-Dokument |
| Datenminimierung | ✅ | Keine lokale Speicherung von Audio und Texten (`data-mapping.md` „Lokale Speicherung“; im Code nur `SaveCache` schreibt eine Datei — Richtlinie, keine personenbezogenen Daten) | Zwischenablage und Zielprogramm sind die einzigen Orte, an denen der Text den Client verlässt |
| Zwischenablage / Verlauf | ✅ (Windows) / ⚠️ | siehe Security | macOS/Linux: Systemrichtlinien/Zwischenablage-Manager beachten |
| Lokale Tokens und UI-Daten dokumentiert | ✅ | `data-mapping.md` Tabelle „Lokale Speicherung“ | Unverschlüsselt (siehe oben) |
| Verschlüsselung in Transit | ❌ (nicht erzwungen) | siehe Security | Betreiberverantwortung |
| Externe Auftragsverarbeiter | ✅ | Client hat keine; nur der konfigurierte Server | Server-Dokument beschreibt LLM/Sanitizer/Objektspeicher |
| Einwilligung/Hinweis zur Nutzung | ✅ | Pflicht-Bestätigung beim ersten Login (`DisclaimerGate.tsx`, `/auth/accept-disclaimer`); [`DISCLAIMER.md`](../DISCLAIMER.md) | Serverseitig protokolliert (`disclaimer_accepted_at`) |
| Betroffenenrechte (Auskunft, Löschung, Übertragbarkeit, Widerspruch) | ➖ | Daten liegen auf dem Server | siehe Server-Checkliste |
| Aufbewahrungsfristen | ✅/➖ | Client: keine Aufbewahrung; Windows-Zwischenablage 30 s (konfigurierbar) | Server: siehe Server-Doku |
| „Bekannte Lücken“ explizit benannt | ✅ | `data-mapping.md` eigener Abschnitt | Lücken werden nicht versteckt |
| Betriebsrat/Mitbestimmung | ➖ | [`DISCLAIMER.md`](../DISCLAIMER.md) weist auf mögliche Mitbestimmungsrechte hin | Entscheidung des Betreibers |

## Lizenz / Compliance

| Prüfpunkt | Status | Fundstelle/Beleg | Kommentar |
|---|---|---|---|
| `LICENSE` vorhanden, vollständig | ✅ | `LICENSE` (voller AGPL-3.0-Text), `LICENSES/AGPL-3.0-or-later.txt` | |
| Datei-Lizenzheader | ✅ | Alle `.go`-Dateien und die Schlüssel-`.tsx` tragen den AGPL-Header; `scripts/add_spdx_headers.py` ergänzt die SPDX-Kurzform mechanisch | Ältere Frontend-Dateien ohne Header sind in `REUSE.toml` einzeln gelistet |
| REUSE-Konformität | ✅ | `reuse lint` grün (198/198 Dateien mit Copyright- und Lizenzinformation, REUSE-Tool 6.2.0, Stand 2026-10-01); `REUSE.toml`, `LICENSES/`; CI-Job „REUSE compliance“ | Direkter Nachweis |
| Third-Party-Notices | ✅ | [`THIRD_PARTY_NOTICES.md`](../THIRD_PARTY_NOTICES.md), generiert mit `scripts/check_licenses.py --write-notices` (30 Go-Module + 5 npm-Pakete + eingebettete Assets) | Regenerierbar, nicht von Hand gepflegt |
| Automatische Lizenzkompatibilitäts-Prüfung | ✅ | `scripts/check_licenses.py` (Go-Module über `go list -deps` für Linux/Windows/macOS, npm nur Produktion, Allowlist AGPL-kompatibler Lizenzen) | Manuell auszuführen; Klassifizierung der Go-Lizenzen ist heuristisch („UNKNOWN“ → von Hand prüfen) |
| Eingebundene Fremdquellen deklariert | ✅ | `third_party/hid` (BSD-3-Clause; hidapi BSD-3-Clause; libusb LGPL-2.1-or-later, unter Linux dynamisch gelinkt), `third_party/hotkey` (MIT) — per `REUSE.toml` annotiert | Annotationen aus den Upstream-Hinweisen abgeleitet; bei Updates prüfen |
| Keine proprietären Binärdateien im Repo | ✅ | Microsoft-WebView2-Bootstrapper (`build/windows/installer/tmp/`) aus Git entfernt und ignoriert; Wails lädt ihn beim NSIS-Build | |
| Schrift-/Logo-Rechte | ⚠️ | Nunito: OFL-1.1 (`frontend/src/assets/fonts/OFL.txt`); Logo/Icons als Projekt-Artwork annotiert | Rechte am Logo vom Maintainer zu bestätigen |
| `AUTHORS` | ✅ | `AUTHORS` | |

## Projekt-Health / Governance

| Prüfpunkt | Status | Fundstelle/Beleg | Kommentar |
|---|---|---|---|
| CI/CD-Pipeline | ✅ | `.github/workflows/ci.yml`: Go (gofmt, vet, Tests unter xvfb), Frontend (Typecheck, Build), REUSE | Neu eingerichtet; erster Lauf auf GitHub steht aus |
| Dependency-Scanning (Dependabot) | ✅ | `.github/dependabot.yml`: wöchentlich `gomod`, `npm` (`frontend/`), GitHub Actions | Kein `govulncheck`/CVE-Gate |
| `CONTRIBUTING.md` | ✅ | [`CONTRIBUTING.md`](../CONTRIBUTING.md) | |
| `CODE_OF_CONDUCT.md` | ✅ | [`CODE_OF_CONDUCT.md`](../CODE_OF_CONDUCT.md) (Contributor Covenant 2.1, adaptiert) | |
| `SECURITY.md` | ✅ | [`SECURITY.md`](../SECURITY.md) | |
| `CHANGELOG.md` | ✅ | [`CHANGELOG.md`](../CHANGELOG.md), Keep-a-Changelog | |
| Issue-/PR-Templates | ✅ | `.github/ISSUE_TEMPLATE/*.yaml`, `.github/PULL_REQUEST_TEMPLATE.md` (inkl. Hinweis „keine Patientendaten“) | |
| Versionierungs-Tooling | ✅ | `scripts/bump_version.py` (Quelle: `wails.json`, synct `frontend/package.json` + Lockfile, verschiebt `[Unreleased]`) | |
| Versionierung tatsächlich genutzt (Git-Tags) | ❌ | noch kein Tag; `.github/workflows/release.yml` prüft Tag = `wails.json`-Version und veröffentlicht die Changelog-Notizen | Erster Tag ausstehend |
| Release-Artefakte | ⚠️ | `scripts/release.sh` lädt lokal gebaute MSI/EXE/ZIP unter festen Namen hoch (der Server verlinkt `releases/latest/download/…`) | Manuell, unsigniert (siehe Security); keine Prüfsummen, keine SBOM, kein CI-Build |
| Testabdeckung | ⚠️ | 28 Go-Tests (`internal/clientpolicy`, Einstellungen, Hold-Tracker, HID-Protokoll); Frontend: kein Test-Setup | OS-nahe Teile (Einfügen, Hotkeys, WebView-Berechtigungen) nur manuell prüfbar |
| Commit-/Maintainer-Aktivität | ✅ | 28 Commits seit 2026-08, laufende Entwicklung | |
| Bus-Faktor | ❌ (=1) | Alle Commits stammen von einer Person (unter zwei Git-Identitäten) | Kein zweiter Maintainer, kein Nachfolgeprozess |
| Build-Reproduzierbarkeit | ⚠️ | `go.sum` und `package-lock.json` vorhanden; Toolchain-Versionen (Wails-CLI, mingw, wixl) nur per Prüfwarnung in `build-windows-msi.sh` | Keine bit-identischen Builds, kein CI-Build |
| Dokumentation | ✅ | [`README.md`](../README.md), [`opendox-client.md`](opendox-client.md), `.agents/AGENTS.md`, `build/windows/installer/DEPLOYMENT.md` | |

## Zusammenfassung

**Stärken:** keine Telemetrie und nur ein einziges Kommunikationsziel (der konfigurierte Server);
keine lokale Ablage von Audio und Texten; explizite Bestätigung vor dem Einfügen; gehärtete
Zwischenablage unter Windows (Verlauf-/Cloud-Ausschluss, Auto-Leeren, optionale Ziel-Allow-Liste);
sauberer SSO-Flow nach RFC 8252; durchgängig AGPL-Header, REUSE-konform, generierte
Abhängigkeitsliste; ehrlich dokumentierte Lücken.

**Offene Punkte (in Reihenfolge der Relevanz für einen Rollout):**

1. Code-Signierung der Windows-Builds (Smart App Control) und Notarisierung für macOS.
2. TLS-Zwang bzw. Warnung bei `http://`; Tokens aus URLs herausnehmen (Server-Änderung nötig).
3. Tokens im Schlüsselbund des Betriebssystems statt `localStorage`.
4. Zwischenablage-Schutz und Einfügeziel-Prüfung für macOS/Linux.
5. Erster Release-Tag, Prüfsummen/SBOM für Release-Dateien, CI-Build der Installer.
6. Zweiter Maintainer (Bus-Faktor 1) und Tests für OIDC-Loopback und Einfügen.
