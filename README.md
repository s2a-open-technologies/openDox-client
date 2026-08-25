# openDox Client

Always-on-top Desktop-Overlay für die Aufnahme und Live-Transkription medizinischer
Diktate. Setzt auf einen separaten [openDox](https://opendox.ai)-Backend-Server auf und
ist als schmales Fenster konzipiert, das während der Arbeit sichtbar bleibt, ohne zu
stören.

Gebaut mit [Wails v2](https://wails.io) — Go-Backend + React/TypeScript-Frontend. Läuft
auf Linux, macOS und Windows.

Teil des [openDox](https://opendox.ai)-Projekts. Dieser Client ist Open Source unter der
**AGPLv3** (siehe [LICENSE](./LICENSE)) — der von ihm angesprochene Server ist ein
separates Produkt und nicht Teil dieses Repositories.

> **Vor dem Einsatz lesen:** openDox ist kein zugelassenes Medizinprodukt. Siehe
> [DISCLAIMER.md](./DISCLAIMER.md).

## Features

- **Live-Transkription** über WebSocket mit rohem PCM-Audio (16 kHz, Mono)
- **Globale Hotkeys** — Aufnahme/Pause/Akzeptieren unabhängig vom fokussierten Fenster
- **Physische Diktiermikro-Tasten** — Philips SpeechMike / Nuance PowerMic lösen
  dieselben Aktionen wie die Hotkeys aus (siehe [Hardware](#diktiermikros))
- **Accept-and-Insert-Workflow** — Transkript wird erst nach expliziter Bestätigung ins
  fokussierte Zielfenster eingefügt; vorher nicht kopier- oder auswählbar
- **Sitzungs-Wiederaufnahme** — abgebrochene/unfertige Aufnahmen können fortgesetzt werden
- **Kategorien** — Aufnahmen einer Kategorie zuordnen, auch nachträglich mit
  automatischer Neu-Zusammenfassung
- **Login per Passwort oder OIDC/SSO** — nativer Loopback-Redirect-Flow, kein
  eingebetteter Login im WebView
- **Auto-Positionierung** oben rechts auf dem primären Display
- **OS-Integration** — Mikrofon-Berechtigungen werden auf Linux (WebKitGTK) und macOS
  (WKWebView) automatisch erteilt, ohne wiederholte Dialoge; native
  Desktop-Benachrichtigung nach dem Einfügen als Prüf-Erinnerung

## Voraussetzungen

### System (Linux)

```bash
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev pkg-config build-essential
```

Zusätzlich für die Desktop-Benachrichtigung nach dem Akzeptieren:

```bash
sudo apt install libnotify-bin
```

### macOS

```bash
brew install go nodejs wails
```

Beim Bauen (`wails build`) und beim Starten (`wails dev`, `go run`) muss auf macOS das
`UniformTypeIdentifiers`-Framework über CGO_LDFLAGS hinzugefügt werden, da `robotgo`
auf `UTType` verweist (sonst: `symbol(s) not found for architecture arm64`):

```bash
CGO_LDFLAGS="-framework UniformTypeIdentifiers" wails build
CGO_LDFLAGS="-framework UniformTypeIdentifiers" wails dev
CGO_LDFLAGS="-framework UniformTypeIdentifiers" go run -tags dev,webkit2_41 .
```

Um das dauerhaft zu setzen, in `~/.zshrc`:

```bash
export CGO_LDFLAGS="-framework UniformTypeIdentifiers"
```

#### Produktions-Build: `./build-macos.sh` statt `wails build`

```bash
./build-macos.sh          # Standard (arm64)
./build-macos.sh --debug  # Debug-Build
```

Baut wie `wails build`, signiert das `.app`-Bundle danach aber zusätzlich. Das ist auf
macOS notwendig, damit die drei Systemberechtigungen unten überhaupt zuverlässig halten
— siehe Kommentar im Skript für Details und wie das kostenlose Signierzertifikat
eingerichtet wird.

#### Erforderliche Systemberechtigungen

openDox braucht auf macOS drei Freigaben unter **Systemeinstellungen → Datenschutz &
Sicherheit**, jeweils für `opendox-client` (bzw. `build/bin/opendox-client.app`):

| Berechtigung | Wofür | Ohne sie |
|---|---|---|
| **Mikrofon** | Audioaufnahme (`getUserMedia`) | Aufnahme startet nicht |
| **Bedienhilfen** (Accessibility) | Text per Cmd+V ins Zielfenster einfügen ("Akzeptieren und einfügen") | Text bleibt nur in der Zwischenablage, kein automatisches Einfügen |
| **Eingabeüberwachung** (Input Monitoring) | Globale Hotkeys (Cmd+Option+R/P/V/H) und Diktiermikro-Tasten | Hotkeys/Tasten lösen nichts aus |

Alle drei sind über den "+"-Button hinzuzufügen (direkt aus dem Finder in die Liste ziehen
statt über den Datei-Auswahldialog zu suchen — vermeidet Verwechslung, falls mehrere
gleichnamige Build-Artefakte im Projekt liegen) und der Schalter muss aktiviert sein.
**Nach jeder Änderung die App komplett beenden (Cmd+Q) und neu starten**, damit macOS die
Freigabe übernimmt.

Wichtig: Ohne echtes, von Apple ausgestelltes Signierzertifikat (siehe `build-macos.sh`)
hält macOS auf aktuellen Versionen diese Freigaben nicht zuverlässig über Rebuilds hinweg
— rein Ad-hoc-signierte Builds können den Schalter in den Einstellungen zwar anzeigen,
ohne dass die App das intern tatsächlich als erteilt erkennt. Das kostenlose
"Apple Development"-Zertifikat (über Xcode, kein bezahltes Developer Program nötig) löst
das zuverlässig.

### Windows

Cross-Build von Linux aus über `./compile.sh` (siehe [Build](#build-produktion)), oder
nativ auf Windows mit `wails build`. `CGO_ENABLED=1` muss gesetzt sein, da `robotgo`
(Tastatur-Injektion) cgo braucht.

### Toolchain

- **Go** ≥ 1.25
- **Node.js** ≥ 18 + npm
- **Wails CLI** v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

> Linux und macOS bauen out-of-the-box (siehe jeweiliger Abschnitt oben). `webview_perms.go`
> nutzt cgo gegen `webkit2gtk-4.1`/`gtk+-3.0` und ist auf Linux beschränkt; macOS hat mit
> `webview_perms_darwin.go` eine eigene Entsprechung.

## Setup

```bash
# Frontend-Abhängigkeiten installieren
cd frontend && npm install && cd ..

# Backend-Abhängigkeiten
go mod download
```

## Entwicklung

### Mit der Wails CLI

```bash
wails dev
```

Startet die App mit Hot-Reloading (Vite Dev-Server). Zusätzlich läuft ein Dev-Server
unter `http://localhost:34115`, über den die Go-Methoden auch im Browser getestet
werden können.

### Direkt über Go (mit Build-Tags)

```bash
go run -tags dev,webkit2_41 .
```

- `dev` — schaltet Wails-Dev-Features frei (Assetserver für das nicht eingebettete
  Frontend, Frontend wird vom Vite-Dev-Server geladen)
- `webkit2_41` — wählt die `webkit2gtk-4.1`-API-Variante auf Linux

> Vorab einmalig `cd frontend && npm run dev` starten, damit Vite das Frontend
> bereitstellt — sonst lädt das Fenster leer.

## Konfiguration

Die Server-URL wird über `settings.json` im Projektstamm konfiguriert:

```json
{
    "server_url": "http://localhost:10000"
}
```

- `server_url` — REST-API-URL (auch für WebSocket verwendet; `http://` wird automatisch
  zu `ws://`, `https://` zu `wss://`)

Beim Start liest das Go-Backend die Datei ein (`//go:embed settings.json` — die Datei
muss zum Bauzeitpunkt existieren) und reicht die Einstellungen über `GetSettings()` ans
Frontend weiter, das sie über `SettingsContext` an allen Stellen konsumiert, die eine
Server-URL brauchen. Ändern der Server-Adresse erfordert also einen Rebuild — die Datei
wird zur Kompilierzeit eingebettet, nicht zur Laufzeit gelesen. Fehlt `settings.json`
oder ist sie ungültig, wird als Default `http://localhost:10000` verwendet.

### Frontend eigenständig

```bash
cd frontend
npm run dev        # Vite Dev-Server
npm run typecheck  # tsc --noEmit
npm run build      # Produktions-Build nach frontend/dist
```

## Build (Produktion)

```bash
wails build
```

Bündelt das Frontend (`npm run build`) und bettet `frontend/dist` über
`//go:embed` in das Go-Binary ein. Auf macOS stattdessen `./build-macos.sh` verwenden
(siehe [macOS](#macos)).

### Alle Plattformen in einem Schritt

```bash
./compile.sh
```

Baut Linux (nativ) und Windows (Cross-Build via `mingw-w64`, inkl. NSIS-Installer über das
Template in `build/windows/installer/`) hintereinander und startet die Apps **nicht**
automatisch. Voraussetzung für den Windows-Build: `sudo apt install gcc-mingw-w64-x86-64 nsis`.

macOS wird von `compile.sh` bewusst ausgelassen — Wails v2 unterstützt kein Cross-Compiling für
macOS-Targets von einem Nicht-Mac-Host. Auf einem Mac nativ bauen, siehe
[Produktions-Build](#produktions-build-build-macossh-statt-wails-build) oben.

## Tastenkürzel

| Linux/Windows | macOS         | Aktion                                    |
|---------------|---------------|--------------------------------------------|
| `Ctrl+Alt+R`  | `Cmd+Option+R` | Aufnahme starten / stoppen                 |
| `Ctrl+Alt+P`  | `Cmd+Option+P` | Aufnahme pausieren / fortsetzen            |
| `Ctrl+Alt+V`  | `Cmd+Option+V` | Transkript akzeptieren & in Zielfenster einfügen |
| `Ctrl+Alt+H`  | `Cmd+Option+H` | Overlay ein-/ausblenden                    |

Die Hotkeys sind systemweit registriert und funktionieren unabhängig davon, welches
Fenster fokussiert ist (auf macOS erfordert das die Eingabeüberwachung-Berechtigung, siehe
oben). Physische Tasten an unterstützten Diktiermikros lösen dieselben drei erstgenannten
Aktionen aus, siehe [Diktiermikros](#diktiermikros).

### Accept-and-Insert-Flow

1. Sprecher beendet das Diktat → Text steht im Overlay, **nicht** in der Zwischenablage.
2. Nutzer klickt **„Akzeptieren und einfügen"** oder drückt den Hotkey/die Diktiermikro-Taste.
3. Text wird in die Zwischenablage gelegt, eine Review-Benachrichtigung erscheint, und
   Cmd/Ctrl+V wird simuliert — der Text erscheint im zuvor fokussierten Zielfenster.

Der Transkript-Text lässt sich nicht manuell kopieren (keine Textauswahl, kein
Kontextmenü); er verlässt die App ausschließlich über den Akzeptieren-Schritt.

## Diktiermikros

Physische Tasten auf unterstützten Philips SpeechMike- und Nuance-PowerMic-Geräten lösen
dieselben Aktionen aus wie die Tastatur-Hotkeys (Aufnahme starten/stoppen, pausieren,
akzeptieren & einfügen) — ohne dass die App im Vordergrund oder fokussiert sein muss.
Gerätetabelle und Protokoll sind aus
[GoogleChromeLabs/dictation_support](https://github.com/GoogleChromeLabs/dictation_support)
(Apache-2.0) portiert.

> **Hinweis:** Die Byte-Offsets für das Tasten-Parsing sind bisher nicht gegen echte
> Hardware verifiziert. Mit `ODX_DICTATION_MIC_DEBUG=1` als Umgebungsvariable protokolliert
> die App die rohen Bytes jedes erkannten Geräts — hilfreich, um das mit einem konkreten
> Gerät zu verifizieren oder ein neues Gerät zu unterstützen.

## Architektur

```
┌──────────────────────────────────────────────────────────────────┐
│  Wails App (main.go)                                              │
│  ┌──────────────────────────┐   ┌───────────────────────────────┐ │
│  │  Go-Backend (app.go, ...) │   │  React-Frontend                │ │
│  │  - HTTP/WS-Proxy          │◄──┤  - Recorder.tsx (Kern)         │ │
│  │  - OS-Integration         │   │  - AuthContext (JWT)           │ │
│  │    (Hotkeys, Clipboard,   │──►│  - SettingsContext             │ │
│  │     Diktiermikro, OIDC)   │   │  - WebSocket-Audio-Pipe        │ │
│  └───────────┬────────────────┘   └───────────────────────────────┘ │
└──────────────┼──────────────────────────────────────────────────────┘
               │  Bearer-Token (vom Frontend durchgereicht)
               ▼
┌───────────────────────────────────────┐
│  openDox Server (settings.json)        │
│  - Auth · Records · Transkription      │
│  - LLM-Zusammenfassung                 │
└─────────────────────────────────────────┘
```

- **Go-Backend** — dünner Proxy + OS-Integration (Fensterposition, Hotkeys,
  Clipboard, WebView-Permissions, Diktiermikro-HID, natives OIDC-Login). Jede
  exportierte Methode auf `App` wird über die generierten Bindings
  (`frontend/wailsjs/`) zur TS-Funktion. Die Go-Seite ist zustandslos bzgl. Auth —
  das Token kommt immer als Parameter vom Frontend.
- **Frontend** — effektiv ein Overlay-View (`App.tsx` → `LoginScreen`/`DisclaimerGate`
  oder `Recorder`). Auth läuft direkt via `fetch`/`authFetch`, alles was Token-seitig
  angefasst werden muss oder OS-Zugriff braucht, geht über die Go-Bindings.

### Aufnahme-Flow (`Recorder.tsx`)

1. `createRecord()` → Session auf dem Server anlegen (oder Resume).
2. `getUserMedia` → `AudioContext` → Inline-`AudioWorkletProcessor` (Float32-PCM)
   → WebSocket ans Backend.
3. Server-Message: `partial` (Live-Text), `text`/`cleanup` (stabil),
   `status: "finished"` (Hintergrund-Task gestartet).
4. Stop → `{event:"stop"}`, dann Polling des Task-Status bis terminal
   (Transkription + LLM-Summary).

## Projektstruktur

```
.
├── main.go                      # Wails-Setup, //go:embed frontend/dist
├── app.go                       # App-Struct, Bound Methods, Hotkeys, OS-Integration
├── oidc.go                      # Nativer OIDC/SSO-Login (Loopback-Redirect)
├── dictation_mic*.go            # Diktiermikro-HID-Protokoll (SpeechMike/PowerMic)
├── webview_perms.go             # Linux: GTK/cgo, erteilt WebKitGTK-Mikrofonrecht auto.
├── webview_perms_darwin.go      # macOS: erspart WKWebViews wiederholtes Mikrofon-Popup
├── accessibility_darwin.go      # macOS: prüft Bedienhilfen-Berechtigung (AXIsProcessTrusted)
├── paste_darwin.go              # macOS: simuliert Cmd+V direkt über CGEventPost
├── hotkey_mod_*.go              # Plattformspezifische Hotkey-Modifier (Ctrl+Alt vs. Cmd+Option)
├── build-macos.sh               # macOS-Produktionsbuild inkl. Signierung
├── settings.json                # Server-URL, zur Bauzeit eingebettet
├── LICENSE                      # AGPLv3, vollständiger Text
├── DISCLAIMER.md                # Kein-Medizinprodukt-Hinweis (Langfassung)
├── wails.json                   # Wails-Projekt-Konfiguration
├── frontend/
│   ├── src/
│   │   ├── App.tsx                    # Root: Login/Disclaimer/Recorder
│   │   ├── components/
│   │   │   ├── Recorder.tsx           # Kern: Aufnahme, WS, Transkript, Controls
│   │   │   ├── LoginScreen.tsx
│   │   │   ├── DisclaimerGate.tsx     # Kein-Medizinprodukt-Bestätigung
│   │   │   └── ui/                    # Button, Modal, VoiceVisualizer
│   │   └── contexts/
│   │       ├── AuthContext.tsx        # JWT in localStorage, /auth/me
│   │       └── SettingsContext.tsx    # Server-URL vom Go-Backend
│   ├── wailsjs/                       # Generierte Bindings (nicht manuell editieren)
│   └── package.json
└── build/                       # Wails Build-Assets (Icons, Manifest, Plattform-Configs)
```

> Nach Änderungen an den Methoden-Signaturen von `App` die Bindings via
> `wails dev`/`wails build` regenerieren — `frontend/wailsjs/` nicht von Hand
> editieren.

Ausführlichere Architektur-Hintergründe (warum bestimmte Entscheidungen so getroffen
wurden) stehen in [`.agents/AGENTS.md`](./.agents/AGENTS.md) — primär für
AI-Coding-Agents geschrieben, aber auch für Menschen lesbar.

## Beitragen

Issues und Pull Requests sind willkommen. Vor größeren Änderungen gerne erst ein Issue
aufmachen, um Richtung und Umfang abzustimmen. Neue oder substanziell überarbeitete
Quelldateien sollten den AGPLv3-Lizenzkopf tragen (siehe bestehende `.go`- und
`.tsx`-Dateien für das Format).

## Lizenz

Copyright (C) 2026 Christopher Abanilla <christopher.abanilla@gmx.de>

Dieses Programm ist Freie Software: Sie können es unter den Bedingungen der
**GNU Affero General Public License v3** (oder, nach Ihrer Wahl, einer späteren Version)
weitergeben und/oder modifizieren. Der vollständige Lizenztext liegt in
[`LICENSE`](./LICENSE) bei; siehe auch
<https://www.gnu.org/licenses/agpl-3.0.html>.

Dieses Programm wird in der Hoffnung bereitgestellt, dass es nützlich ist, jedoch
**ohne jegliche Gewährleistung** — auch ohne die stillschweigende Gewährleistung der
Marktgängigkeit oder Eignung für einen bestimmten Zweck. Siehe die GNU Affero General
Public License für weitere Details.

Siehe auch [DISCLAIMER.md](./DISCLAIMER.md) für den medizinisch-regulatorischen Hinweis.
