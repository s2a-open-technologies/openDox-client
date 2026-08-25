# AGENTS.md

Repo-specific guidance for AI coding agents working in this codebase.

## What this is

openDox client — a Wails v2 desktop app (Go backend + React/TypeScript frontend) that acts as a small
always-on-top overlay for recording and live-transcribing medical dictations against a separate openDox
backend server. The Go side is a thin HTTP/WebSocket proxy plus OS integration (window positioning,
global hotkeys, clipboard/keystroke injection, WebView permissions, native OIDC login); almost all
UI/business logic lives in the React frontend.

## Commands

Run from the repo root unless noted.

- `wails dev` — live development with hot-reloading Vite frontend. Dev server at `http://localhost:34115`
  for browser debugging with access to bound Go methods.
- `wails build` — production build (runs `npm run build`, embeds `frontend/dist` via `//go:embed` in
  `main.go`).
- `./build-macos.sh` — macOS production build wrapper: runs `wails build`, then signs the .app bundle
  with a local "Apple Development" identity if one exists in the keychain (falls back to ad-hoc `-`
  otherwise). Use this instead of `wails build` on macOS. Empirically (macOS 26.4/arm64), pure ad-hoc or
  self-signed signatures do **not** reliably persist Accessibility/Input-Monitoring TCC grants across
  rebuilds even with a stable bundle ID — `AXIsProcessTrusted()` kept returning false at runtime despite
  the toggle showing enabled in System Settings. A real Apple-issued certificate fixed it. The free
  "Apple Development" certificate (Xcode → Settings → Accounts → sign in → Manage Certificates → +) works
  fine, no paid Developer Program membership needed — see comment header in `build-macos.sh` for full
  setup steps, including recovering from a missing/expired WWDR intermediate cert
  (`security find-identity -v -p codesigning` reporting 0 valid identities despite the cert being present
  is the symptom).
- `go run -tags dev,webkit2_41 .` — run directly via Go instead of the Wails CLI. The `dev` tag serves the
  frontend from the Vite dev server rather than the embedded bundle, so run `cd frontend && npm run dev`
  first or the window loads blank. `webkit2_41` selects the `webkit2gtk-4.1` API variant on Linux.
- Frontend only, from `frontend/`:
  - `npm run dev` — Vite dev server.
  - `npm run build` — Vite production build (just `vite build`, **no** `tsc`).
  - `npm run typecheck` — `tsc --noEmit`. This is the only compile check; run it after frontend changes.
- No lint or test setup exists for either side — don't assume `npm test`, `go test`, or a linter.

## Build dependencies and platform stubs

Several OS-specific files are split into a real implementation plus no-op stubs via `//go:build` tags —
follow this pattern (rather than adding runtime `runtime.GOOS` checks) when adding new OS-specific
behavior:

- `webview_perms.go` (`//go:build linux`) uses cgo against `webkit2gtk-4.1` and `gtk+-3.0` (via
  `pkg-config`) to auto-grant mic/camera permissions; `webview_perms_darwin.go` (`//go:build darwin`) does
  the macOS equivalent (see WebView permissions below); `webview_perms_other.go`
  (`//go:build !linux && !darwin`) is the no-op stub for Windows, where WebView2 doesn't need this.
- `accessibility_darwin.go` (`//go:build darwin`) uses cgo/`ApplicationServices` to silently check
  `AXIsProcessTrusted()` (see Accessibility below); `accessibility_stub.go` (`//go:build !darwin`) always
  returns `true`.
- `paste_darwin.go` (`//go:build darwin`) simulates Cmd+V via a hardcoded-keycode `CGEventPost` call (see
  Clipboard / accept-and-insert below); `paste_stub.go` (`//go:build !darwin`) is a no-op, since
  Linux/Windows type the text directly instead of pasting.

`dictation_mic.go` uses `github.com/karalabe/hid`, which on Linux is backed by `libusb` via cgo —
needs `libusb-1.0-0-dev` (`pkg-config libusb-1.0`) installed to build. Windows/macOS builds of this
dependency don't need an extra system package.

### macOS accessibility trust (`accessibility_darwin.go`)

`golang.design/x/hotkey` (global hotkeys, needs Input Monitoring) and `paste_darwin.go`'s `CGEventPost`
call (needs Accessibility) both require the process to be trusted on macOS. `App.startup` calls
`ensureAccessibility()` once at launch and just logs if untrusted — it deliberately does **not** trigger
the system prompt dialog, to avoid nagging on every rebuild. `CheckAccessibility()`/`CheckHotkeys()` are
bound to the frontend so it can show an in-app toast pointing the user at System Settings instead. See
Commands above for why `./build-macos.sh` (a real Apple-issued signing identity, not just a stable ad-hoc
bundle ID) is required for these grants to actually be honored at runtime.

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

### Server URL configuration

The backend URL is **not** hardcoded on the Go side: `settings.json` (repo root) is embedded via
`//go:embed settings.json` in `main.go` and read by `loadSettings()` into `App.opendoxURL`/`wsURL` at
startup (falls back to `http://localhost:10000` if the file is missing/unparsable). The bound
`GetSettings()` method exposes it to the frontend, where `SettingsContext.tsx` fetches it once on mount;
`AuthContext.tsx`, `Recorder.tsx`, and `LoginScreen.tsx` all read `settings?.server_url` (falling back to
the same `http://localhost:10000` default) for their REST/WS calls — a single source, no separately
hardcoded frontend constant. Changing the server requires editing `settings.json` and rebuilding —
`//go:embed` reads it at compile time, not runtime — and the file must exist for `wails build`/`go run`
to compile at all.

### Auth flow

`AuthContext.tsx` stores the JWT in `localStorage` (`access_token`) and re-fetches `/api/v1/auth/me` on
mount. Bound Go methods take the token as an explicit parameter (from `getToken()`) — the Go side is
stateless wrt auth.

`LoginWithOIDC()` (`oidc.go`) is the alternative to password login: an RFC 8252 native loopback flow. It
opens a random-port `127.0.0.1` listener, builds a per-attempt random `callbackPath` token (so no other
local process can guess the callback URL and inject tokens), opens the system browser to the server's
OIDC login URL, and returns immediately without waiting for the user to finish in the browser. The result
arrives later as a Wails event — `oidc:login-complete` (with access/refresh tokens) or `oidc:login-error`
— consumed wherever `AuthContext`/`LoginScreen.tsx` listens for it. Has a 5-minute timeout and an
in-flight guard (`oidcLoginInFlight`) so only one attempt runs at a time.

### Global hotkeys (cross-boundary wiring)

System-wide hotkeys are registered in Go (`registerGlobalHotkeys` in `app.go`) using
`golang.design/x/hotkey` and emitted to the frontend as Wails events consumed via `EventsOn` in
`Recorder.tsx`:

- `Ctrl+Alt+R` → `hotkey:toggle-record` (start/stop)
- `Ctrl+Alt+P` → `hotkey:toggle-pause` (pause/resume)
- `Ctrl+Alt+V` → `hotkey:accept-insert` (accept transcript + paste)
- `Ctrl+Alt+H` → toggles the overlay window's visibility (`registerWindowHotkey` in `app.go`); no
  frontend event, handled entirely in Go via `runtime.WindowHide`/`WindowShow`. While hidden, the
  three hotkeys above (and the dictation-mic buttons below) are suppressed via `a.emitHotkeyEvent`.

`Mod1` maps to **Alt** on this X11 setup (verified via `xmodmap -pm`; the `x/hotkey` modifier mapping
isn't platform-guaranteed). Keydown is debounced (~700ms) against X11 auto-repeat. Modifier constants and
the platform-conventional combo (Ctrl+Alt vs. Cmd+Option on macOS) live in `hotkey_mod_*.go`, one per
platform, selected by Go build tags.

`registerHotkey`/`registerWindowHotkey` count failed `hk.Register()` calls in `App.hotkeyFailures`; the
bound `CheckHotkeys()` reports whether any hotkey failed to register, so the frontend can surface a
warning instead of leaving the user to wonder why a hotkey silently does nothing (most commonly a missing
macOS Input Monitoring grant — see Accessibility above).

Closure gotcha: the `EventsOn` listeners subscribe once on mount and would otherwise capture stale state.
`Recorder.tsx` keeps latest values in refs (`isRecordingRef`, `startRecordingRef`,
`stableTranscriptRef`, `acceptAndInsertRef`, …) synced every render — follow this pattern when adding
hotkey-driven state.

### Dictation mic buttons (`dictation_mic.go`, `dictation_mic_protocol.go`)

Physical buttons on Philips SpeechMike / Nuance PowerMic devices emit the exact same
`eventToggleRecord`/`eventTogglePause`/`eventAcceptInsert` events as the keyboard hotkeys above,
through the shared `a.emitHotkeyEvent` helper — no frontend changes needed to support them.
Device VID/PID table and report-parsing byte offsets are ported from
`GoogleChromeLabs/dictation_support` (Apache-2.0); see
`docs/superpowers/specs/2026-08-06-dictation-mic-design.md` for the full protocol reference and
known risks. **These offsets are unverified against real hardware.** Set
`ODX_DICTATION_MIC_DEBUG=1` to log every raw input report's hex bytes for any matched device —
use this to confirm the offsets once a physical device is available, before trusting button
behavior in production.

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

Transcript text does **not** auto-copy after processing (`CopyToClipboard` exists on `App` but nothing in
the frontend calls it). It reaches the focused window only via the explicit accept step —
`acceptAndInsert()` in `Recorder.tsx`, triggered by the "Akzeptieren und einfügen" button
(`viaHotkey=false`) or the `Ctrl+Alt+V` hotkey/dictation-mic button (`viaHotkey=true`) — which calls
`InsertTranscript(text, viaHotkey)` in `app.go`. The clipboard is always set as a silent fallback/backup,
but the primary path is platform-split:

- **macOS**: `simulatePaste()` (`paste_darwin.go`) simulates Cmd+V against the clipboard just set, via a
  hardcoded-virtual-keycode `CGEventPost` call. **Do not swap this for `robotgo.KeyTap("v", "command")`** —
  that crashes the whole process with a SIGTRAP inside cgo, reproducibly, on this macOS/robotgo
  combination (confirmed both with and without genuine Accessibility trust — see
  [go-vgo/robotgo#690](https://github.com/go-vgo/robotgo/issues/690)). Root cause per
  [go-vgo/robotgo#341](https://github.com/go-vgo/robotgo/issues/341): `KeyTap`'s char→keycode resolution
  calls Carbon TIS/TSM APIs, which Apple documents as main-thread-only, while `InsertTranscript` runs on a
  Wails dispatcher goroutine. `paste_darwin.go` sidesteps this entirely by skipping character resolution
  (Cmd+V's keycode is always the same, layout-independent) and calling `CGEventPost` directly, which has
  no such thread restriction.
- **Linux/Windows**: `robotgo.TypeStr(text)` types character-by-character. A simulated Ctrl+V was tried
  here first but collided with the still-releasing `Ctrl+Alt` of the global hotkey under X11 and left the
  text stranded in the clipboard; a synthetic modifier-release (`KeyUp`) was also tried and reverted (see
  commit `83d1522`) because it scrambled Shift/CapsLock state on non-standard keyboard mappings. A plain
  `300ms` sleep before typing/pasting is the current fix for both issues.

`viaHotkey` also controls window hide/show: a button click focuses the openDox overlay itself (any click
focuses the clicked window), so `InsertTranscript` briefly hides the overlay to return focus to the true
target window, sleeps, types/pastes, then re-shows and repositions it. The hotkey/dictation-mic path never
focused the overlay in the first place, so it skips hide/show entirely — see commits `d8cc475`/`524c706`
for why (hiding unconditionally caused mispositioning and unnecessary flicker on that path).

The transcript is non-selectable (`select-none` + `onContextMenu` suppressed) so it can't leave the app
any other way. Don't re-introduce an automatic clipboard copy after processing.

### Categories

Selected category persists per-user in `localStorage` (`selected_category_id_<userId>`). On load, if the
stored id isn't in the fetched list it falls back to the server category named `Allgemein`. There is no
"no category" option in the UI — `CreateRecord` still treats id `0` as null server-side, but the dropdown
always defaults to a real category.

### WebView permissions (`webview_perms.go`, `webview_perms_darwin.go`)

Both auto-grant `getUserMedia` (mic/camera) permission prompts at the native webview level, called once
from `domReady` (`setupWebviewPermissions()`), but for different reasons and via different mechanisms:

- **Linux** (`webview_perms.go`): auto-grants at the GTK/cgo level so recording never shows an OS dialog;
  every other WebKit permission type (geolocation, notifications, ...) is denied. Idempotent (guarded by a
  `perms-connected` GObject data key), walks the whole GTK widget tree — necessary because Wails doesn't
  expose the underlying `WebKitWebView`.
- **macOS** (`webview_perms_darwin.go`): a *different* problem than Linux — the macOS-level Microphone TCC
  grant is separate from WKWebView's own internal per-origin media-capture permission, and per WebKit's
  documented behavior (Wails v2.15.0's `WailsContext.m` doesn't implement the relevant `WKUIDelegate`
  method, so WebKit falls back to its default) that internal grant is **not** remembered across app
  launches — every `getUserMedia()` call reprompts with WebKit's own popup, independent of TCC state.
  Fixed by injecting the missing delegate method (`webView:requestMediaCapturePermissionForOrigin:...`)
  onto Wails' `WailsContext` class at runtime via the Objective-C runtime (`class_addMethod` +
  `imp_implementationWithBlock`), since we can't patch Wails' vendored source — always grants. Skips
  itself if a future Wails version adds the method natively (checks `class_getInstanceMethod` first).

## Frontend conventions

- Path alias `@/*` → `frontend/src/*` (configured in both `vite.config.js` and `tsconfig.json`).
- UI strings and comments are largely German (German-market medical product) — match the existing language
  when editing nearby copy/comments.
- No CSS framework — plain per-component `.css` imported directly into its `.tsx`.
- `color-scheme: dark` in `style.css` is load-bearing for native form controls (`<select>` + its option
  popup, scrollbars) on WebKitGTK — removing it reintroduces a white-on-white dropdown. Don't drop it.
## Licensing (AGPLv3, open source)

Every `.go` file and several key `.tsx` files (`AuthContext.tsx`, `Recorder.tsx`,
`SettingsContext.tsx`, `DisclaimerGate.tsx`, `voice-visualizer.tsx`) carry an AGPLv3
license header — keep it when substantially rewriting a file, and add it to genuinely new
source files (copy the format from an existing file of the same language; Go uses `//`
line comments, not `/* */`, specifically so the header doesn't break `//go:build`
constraint recognition in files that have one — the constraint must be preceded only by
blank lines and other line comments, see the Go spec). Full license text is in
[`LICENSE`](../LICENSE); the medical/regulatory disclaimer referenced by
`DisclaimerGate.tsx` is in [`DISCLAIMER.md`](../DISCLAIMER.md).

