# AGENTS.md

Repo-specific guidance for AI coding agents working in this codebase.

## What this is

openDox client — a Wails v2 desktop app (Go backend + React/TypeScript frontend) that acts as a small
always-on-top overlay for recording and live-transcribing medical dictations against a separate openDox
backend server (`http://localhost:20000`, hardcoded in `frontend/src/api/index.ts` and `app.go`). The Go
side is a thin HTTP/WebSocket proxy plus OS integration (window positioning, global hotkeys,
clipboard/keystroke injection, WebView permissions); almost all UI/business logic lives in the React
frontend.

## Commands

Run from the repo root unless noted.

- `wails dev` — live development with hot-reloading Vite frontend. Dev server at `http://localhost:34115`
  for browser debugging with access to bound Go methods.
- `wails build` — production build (runs `npm run build`, embeds `frontend/dist` via `//go:embed` in
  `main.go`).
- `go run -tags dev,webkit2_41 .` — run directly via Go instead of the Wails CLI. The `dev` tag serves the
  frontend from the Vite dev server rather than the embedded bundle, so run `cd frontend && npm run dev`
  first or the window loads blank. `webkit2_41` selects the `webkit2gtk-4.1` API variant on Linux.
- Frontend only, from `frontend/`:
  - `npm run dev` — Vite dev server.
  - `npm run build` — Vite production build (just `vite build`, **no** `tsc`).
  - `npm run typecheck` — `tsc --noEmit`. This is the only compile check; run it after frontend changes.
- No lint or test setup exists for either side — don't assume `npm test`, `go test`, or a linter.

## Linux build dependency

`webview_perms.go` uses cgo against `webkit2gtk-4.1` and `gtk+-3.0` (via `pkg-config`) with **no Go build
tag**, so it compiles unconditionally. In practice this only builds on Linux even though `build/darwin`
and `build/windows` assets exist — building on macOS/Windows fails unless this file gets a
`//go:build linux` constraint (and a no-op stub for other platforms).

`dictation_mic.go` uses `github.com/karalabe/hid`, which on Linux is backed by `libusb` via cgo —
needs `libusb-1.0-0-dev` (`pkg-config libusb-1.0`) installed to build. Windows/macOS builds of this
dependency don't need an extra system package.

## Architecture

### Two communication paths

`App` (in `app.go`) is the single struct bound to the frontend (`Bind: []interface{}{app}` in `main.go`).
Every exported method becomes a callable TS function via generated bindings in
`frontend/wailsjs/go/main/App.{js,d.ts}` — regenerate by re-running `wails dev`/`wails build`, **don't
hand-edit `frontend/wailsjs/`**.

The frontend talks to the openDox server two ways:

1. Directly via `fetch`/`authFetch` for pure REST reachable from the browser (auth in `AuthContext.tsx`).
2. Through Go-bound methods (`GoCreateRecord`, `GoGetWSURL`, `GoInsertTranscript`, etc.) for anything
   needing a bearer token attached server-side or OS-level access.

Decision rule when adding a backend call: pure browser REST → `AuthContext`/fetch; needs Go-side token or
an OS capability → new bound method on `App`.

### Auth flow

`AuthContext.tsx` stores the JWT in `localStorage` (`access_token`) and re-fetches `/api/v1/auth/me` on
mount. Bound Go methods take the token as an explicit parameter (from `getToken()`) — the Go side is
stateless wrt auth.

### Global hotkeys (cross-boundary wiring)

System-wide hotkeys are registered in Go (`registerGlobalHotkeys` in `app.go`) using
`golang.design/x/hotkey` and emitted to the frontend as Wails events consumed via `EventsOn` in
`Recorder.tsx`:

- `Ctrl+Alt+R` → `hotkey:toggle-record` (start/stop)
- `Ctrl+Alt+P` → `hotkey:toggle-pause` (pause/resume)
- `Ctrl+Alt+A` → `hotkey:accept-insert` (accept transcript + paste)

`Mod1` maps to **Alt** on this X11 setup (verified via `xmodmap -pm`; the `x/hotkey` modifier mapping
isn't platform-guaranteed). Keydown is debounced (~700ms) against X11 auto-repeat.

Closure gotcha: the `EventsOn` listeners subscribe once on mount and would otherwise capture stale state.
`Recorder.tsx` keeps latest values in refs (`isRecordingRef`, `startRecordingRef`,
`stableTranscriptRef`, `acceptAndInsertRef`, …) synced every render — follow this pattern when adding
hotkey-driven state.

### Recording flow (`Recorder.tsx`)

The core feature. Read this before touching audio/session code:

1. `createRecord()` → `GoCreateRecord` creates a session (or reuses `resumeSessionId`; the server
   `session_store` holds live state across reconnects).
2. `openWebSocket()` → `GoGetWSURL` (token embedded in query string) → raw WebSocket. Messages:
   `partial` (live text), `text`/`cleanup` (stable transcript), `status`.
3. Audio: `getUserMedia` → `AudioContext` → inline `AudioWorkletProcessor` (a string blob at runtime,
   **not** a separate file) forwarding raw Float32 PCM over the socket. `AnalyserNode` feeds the
   visualizers.
4. Stop sends `{event:"stop"}`. The WS `status:"finished"` message **only signals the background task
   (transcribe + LLM summary) has started** — `pollTaskStatus` then polls `/records/status` until
   terminal, and the processing spinner clears there (or on a 90s safety timeout). Don't treat WS
   "finished" or socket close as "recording done".
5. Unmount without explicit stop tears down capture and closes the socket as a plain disconnect — the
   backend keeps the session for resumption. Don't conflate "socket closed" with "finished".

### Clipboard / accept-and-insert

Transcript text does **not** auto-copy after processing. It reaches the focused window only via the
explicit accept step — `InsertTranscript` in `app.go` types the text directly via `robotgo.TypeStr`
(the clipboard is set silently as a fallback), fires the review notification — triggered by the
"Akzeptieren und einfügen" button or `Ctrl+Alt+A`. Direct typing replaced an earlier clipboard +
simulated `Ctrl+V` approach, which was unreliable under X11 because the simulated paste collided with
the still-released `Ctrl+Alt` modifier of the global hotkey and left the text only in the clipboard.
The transcript is non-selectable (`select-none` + `onContextMenu` suppressed) so it can't leave the app
any other way. Don't re-introduce an automatic clipboard copy after processing.

### Categories

Selected category persists per-user in `localStorage` (`selected_category_id_<userId>`). On load, if the
stored id isn't in the fetched list it falls back to the server category named `Allgemein`. There is no
"no category" option in the UI — `CreateRecord` still treats id `0` as null server-side, but the dropdown
always defaults to a real category.

### WebView permissions (`webview_perms.go`)

Auto-grants only microphone/camera (`getUserMedia`) permission prompts at the GTK/cgo level so recording
never shows an OS dialog; every other WebKit permission type (geolocation, notifications, ...) is denied.
Runs once in `domReady`, idempotent (guarded by a `perms-connected` GObject data key), walks the whole GTK
widget tree — necessary because Wails doesn't expose the underlying `WebKitWebView`.

## Frontend conventions

- Path alias `@/*` → `frontend/src/*` (configured in both `vite.config.js` and `tsconfig.json`).
- UI strings and comments are largely German (German-market medical product) — match the existing language
  when editing nearby copy/comments.
- No CSS framework — plain per-component `.css` imported directly into its `.tsx`.
- `color-scheme: dark` in `style.css` is load-bearing for native form controls (`<select>` + its option
  popup, scrollbars) on WebKitGTK — removing it reintroduces a white-on-white dropdown. Don't drop it.
- Files carrying the AGPLv3 header comment (`AuthContext.tsx`, `Recorder.tsx`) keep it if substantially
  rewritten.
