# openDox Client

Always-on-top Desktop-Overlay für die Aufnahme und Live-Transkription medizinischer
Diktate. Setzt auf einen separaten [openDox](https://opendox.ai)-Backend-Server
(`http://localhost:20000`) auf und ist als schmales Fenster konzipiert, das während der
Arbeit sichtbar bleibt, ohne zu stören.

Gebaut mit [Wails v2](https://wails.io) — Go-Backend + React/TypeScript-Frontend.

## Features

- **Live-Transkription** über WebSocket mit rohem PCM-Audio (16 kHz, Mono)
- **Globale Hotkeys** — Aufnahme/Pause/Akzeptieren unabhängig vom fokussierten Fenster
- **Accept-and-Insert-Workflow** — Transkript wird erst nach expliziter Bestätigung in
  die Zwischenablage freigegeben und per `Ctrl+V` in das Zielfenster eingefügt
- **Sitzungs-Wiederaufnahme** — abgebrochene/unfertige Aufnahmen können fortgesetzt werden
- **Auto-Positionierung** oben rechts auf dem primären Display
- **OS-Integration** — Mikrofon-Berechtigungen werden auf GTK-Ebene auto-erteilt
  (kein Dialog), native Desktop-Benachrichtigung nach dem Kopieren

## Voraussetzungen

### System (Linux)

```bash
sudo apt install libwebkit2gtk-4.1-dev libgtk-3-dev pkg-config build-essential
```

Zusätzlich für die Desktop-Benachrichtigung nach dem Akzeptieren:

```bash
sudo apt install libnotify-bin
```

### Toolchain

- **Go** ≥ 1.25
- **Node.js** ≥ 18 + npm
- **Wails CLI** v2: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

> Derzeit wird nur Linux out-of-the-box gebaut. `webview_perms.go` nutzt cgo gegen
> `webkit2gtk-4.1` und `gtk+-3.0`.

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
`//go:embed` in das Go-Binary ein.

### Alle Plattformen in einem Schritt

```bash
./compile.sh
```

Baut Linux (nativ) und Windows (Cross-Build via `mingw-w64`, inkl. NSIS-Installer über das
Template in `build/windows/installer/`) hintereinander und startet die Apps **nicht**
automatisch. Voraussetzung für den Windows-Build: `sudo apt install gcc-mingw-w64-x86-64 nsis`.

macOS wird von `compile.sh` bewusst ausgelassen — Wails v2 unterstützt kein Cross-Compiling für
macOS-Targets von einem Nicht-Mac-Host. Auf einem Mac stattdessen nativ bauen:

```bash
wails build -platform darwin/arm64
```

## Tastenkürzel

| Kürzel        | Aktion                                            |
|---------------|--------------------------------------------------|
| `Ctrl+Alt+R`  | Aufnahme starten / stoppen                       |
| `Ctrl+Alt+P`  | Aufnahme pausieren / fortsetzen                  |
| `Ctrl+Alt+A`  | Transkript akzeptieren & in Zielfenster einfügen |

Die Hotkeys sind systemweit registriert und funktionieren unabhängig davon, welches
Fenster fokussiert ist.

### Accept-and-Insert-Flow

1. Sprecher beendet das Diktat → Text steht im Overlay, **nicht** in der Zwischenablage.
2. Nutzer klickt **„Akzeptieren und einfügen"** oder drückt `Ctrl+Alt+A`.
3. → Text wird in die Zwischenablage gelegt, eine Review-Benachrichtigung erscheint
   und `Ctrl+V` wird simuliert — der Text erscheint im fokussierten Fenster.

Der Transkript-Text lässt sich nicht manuell kopieren (keine Textauswahl, kein
Kontextmenü); er verlässt die App ausschließlich über den Akzeptieren-Schritt.

## Architektur

```
┌─────────────────────────────────────────────────────────────┐
│  Wails App (main.go)                                        │
│  ┌────────────────────────┐   ┌──────────────────────────┐  │
│  │  Go-Backend (app.go)   │   │  React-Frontend          │  │
│  │  - HTTP/WS-Proxy       │◄──┤  - Recorder.tsx (Kern)   │  │
│  │  - OS-Integration      │   │  - AuthContext (JWT)     │  │
│  │  - Globale Hotkeys     │──►│  - WebSocket-Audio-Pipe  │  │
│  │  - Clipboard/Ctrl+V    │   │                          │  │
│  └───────────┬────────────┘   └──────────────────────────┘  │
└──────────────┼──────────────────────────────────────────────┘
               │  Bearer-Token (vom Frontend durchgereicht)
               ▼
┌──────────────────────────────────┐
│  openDox Server (:20000)         │
│  - Auth · Records · Transkription│
│  - LLM-Zusammenfassung           │
└──────────────────────────────────┘
```

- **Go-Backend** — dünner Proxy + OS-Integration (Fensterposition, Hotkeys,
  Clipboard, WebView-Permissions). Jede exportierte Methode auf `App` wird über die
  generierten Bindings (`frontend/wailsjs/`) zur TS-Funktion. Die Go-Seite ist
  zustandslos bzgl. Auth — das Token kommt immer als Parameter vom Frontend.
- **Frontend** — effektiv ein Overlay-View (`App.tsx` → `LoginScreen` oder
  `Recorder`). Auth läuft direkt via `fetch`/`authFetch`, alles was Token-seitig
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
├── main.go              # Wails-Setup, //go:embed frontend/dist
├── app.go               # App-Struct, Bound Methods, Hotkeys, OS-Integration
├── webview_perms.go     # GTK/cgo: auto-erteilt WebKitGTK-Berechtigungen
├── wails.json           # Wails-Projekt-Konfiguration
├── frontend/
│   ├── src/
│   │   ├── App.tsx              # Root: Login/Disclaimer/Recorder
│   │   ├── components/
│   │   │   ├── Recorder.tsx     # Kern: Aufnahme, WS, Transkript, Controls
│   │   │   ├── LoginScreen.tsx
│   │   │   └── ui/              # Button, Modal, VoiceVisualizer
│   │   ├── contexts/AuthContext.tsx  # JWT in localStorage, /auth/me
│   │   └── api/index.ts         # BASE_URL (http://localhost:20000)
│   ├── wailsjs/                 # Generierte Bindings (nicht manuell editieren)
│   └── package.json
└── build/                       # Wails Build-Assets (Icons, Manifest)
```

> Nach Änderungen an den Methoden-Signaturen von `App` die Bindings via
> `wails dev`/`wails build` regenerieren — `frontend/wailsjs/` nicht von Hand
> editieren.

## Lizenz

Copyright (C) 2026 Christopher Abanilla <christopher.abanilla@gmx.de>

Dieses Programm ist Freie Software: Sie können es unter den Bedingungen der
**GNU Affero General Public License v3** weitergeben und/oder modifizieren.
Die vollständige Lizenz findet sich unter
<https://www.gnu.org/licenses/agpl-3.0.html>.

