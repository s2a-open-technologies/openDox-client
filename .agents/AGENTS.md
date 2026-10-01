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

The backend URL is **not** hardcoded on the Go side and is **not** compiled in — it's read at runtime
from a `settings.json`. `loadSettings()` in `app.go` checks, first match wins:

1. The same folder as the executable (`exeDirSettingsPath()` in `settings_path.go`), e.g.
   `C:\Program Files\openDox\settings.json` or `build/bin/settings.json`. For a macOS bundle this is the
   folder containing the `.app`, never inside it (a file in the bundle would break its codesign signature).
   Note `go run` builds into a temp dir, so there this lookup never hits.
2. The per-user config dir (`os.UserConfigDir()`, `runtimeSettingsPath()`):

   - macOS: `~/Library/Application Support/opendox-client/settings.json`
   - Windows: `%AppData%\opendox-client\settings.json`
   - Linux: `~/.config/opendox-client/settings.json` (or `$XDG_CONFIG_HOME`)

A file only counts if it parses and has a non-empty `server_url`; otherwise the next location is tried.
`loadSettings()` reads `{"server_url": "..."}` at startup into `App.opendoxURL`/`wsURL`,
falling back to `http://localhost:10000` if the file is missing or unparsable — so local dev needs no
setup. The bound `GetSettings()` method exposes it to the frontend, where `SettingsContext.tsx` fetches
it once on mount; `AuthContext.tsx`, `Recorder.tsx`, and `LoginScreen.tsx` all read `settings?.server_url`
(same `http://localhost:10000` fallback) for their REST/WS calls — a single source, no separately
hardcoded frontend constant. This is deliberately runtime-only, not `//go:embed`-based: the same build
gets shipped to every customer, and the server address is set per-install by dropping a `settings.json`
next to the executable — no rebuild per customer.

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
- `Ctrl+Alt+D` (hold) → push-to-talk: Keydown emits `hotkey:mic-record-down`, Keyup `hotkey:mic-record-up`
  (`registerHoldHotkeyTimeout`, `holdTracker` swallows auto-repeat) — the same events and frontend path as
  the dictation mic's Record button in hold mode (see below), so it doubles as a hardware-free test. Always
  on, independent of `dictation_mic_mode`. Note: Ctrl+Alt+D is "show desktop" on e.g. Xfce; registration
  then fails (logged, counted in `hotkeyFailures`).
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
known risks. **Hold mode (default).** With `dictation_mic_mode` `"hold"` (settings.json, `"toggle"` = old behavior;
`clientpolicy.Settings.MicHoldMode`), the Record button is push-to-talk: `readDictationMicLoop` emits
`hotkey:mic-record-down` on the rising edge and `hotkey:mic-record-up` on the falling edge (`emitMicRelease`
deliberately ignores overlay visibility and the debounce, and also fires when the device disconnects while
held, so a recording can never get stuck). `Recorder.tsx` starts with `startRecording({autoInsert: true})`;
`pollTaskStatus` then calls `acceptAndInsert(true, text)` directly with the fresh server text (allow-list
prompt still applies). A release before the async start finished is handled via `micReleasedEarlyRef`. The
Stop/Rewind/Ins-Ovr buttons and keyboard hotkeys are unchanged.

**These offsets are unverified against real hardware.** Set
`ODX_DICTATION_MIC_DEBUG=1` to log every raw input report's hex bytes for any matched device —
use this to confirm the offsets once a physical device is available, before trusting button
behavior in production.

### Server-managed client policy (`policy_sync.go`)

The policy keys of settings.json (`clipboard_*`, `insert_allowlist`, `dictation_mic_mode`, `live_view`) are
managed centrally on the openDox server (admin UI → Clients, server repo `~/python/project_red`,
`GET /api/v1/settings/client-config/`, unauthenticated). `syncPolicyLoop` fetches it at startup and every
10 min; `clientpolicy.Merge` lets every field the server sets win over the local file (an explicitly empty
server allowlist turns a local one off). The last server state is cached per server URL
(`os.UserCacheDir()/opendox-client/client-policy-cache.json`) and applied in `NewApp` without network. The
policy can change at runtime: read it only via `a.currentPolicy()` (never cache it in a long-lived variable
— `readDictationMicLoop` reads `MicHoldMode()` per key press), and the frontend re-reads
`GetLiveViewDefault()` on the `policy:changed` event. Installers are not stored on the server: it links to
`releases/latest/download/<fixed name>` of this repo, uploaded by `scripts/release.sh` — keep those
names in sync with `_CLIENT_DOWNLOAD_FILES` in the server's `settings_routes.py`.

### Live view off / compact overlay (`live_view`)

`live_view` in settings.json (`clientpolicy.Settings.LiveViewDefault`, exposed via the bound
`GetLiveViewDefault()` — deliberately **not** in the frontend-bound `Settings` struct, the same JSON key in
two embedded structs would make Go drop both) is only the *default*; the user's eye-icon toggle in the
overlay is stored in `localStorage` (`live_view`) and wins. `Recorder.tsx` derives
`liveEnabled = serverLive && liveView` (`serverLive` = the server-wide `live_backend !== "disabled"`, which
already had an orb-only UI). When off: orb instead of text, `GoGetWSURL(..., live=false)` appends `live=0`
so the server skips live ASR (Parakeet) for that session (audio is still written, final transcription after
stop is unchanged). The window size deliberately stays the same (a compact mode was tried and was disruptive).
The minimize button was removed again: while minimized, the task polling (and thus
auto-insert) stalled until the window was restored — presumably WebView2 background throttling; an attempt to
turn that off via WebView2 browser flags was reverted. Use Ctrl+Alt+H (hide) instead. Ctrl+Alt+H still calls
`WindowUnminimise` before `WindowShow`, in case the window was minimized via the taskbar.

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

**Clipboard hardening and insert target (Windows).** (The Linux/Windows bullet above is historical: on Windows the
text is now pasted with Ctrl+V and only typed in Citrix clients, see `insert_windows.go`; Linux uses simulated Ctrl+V.)
The clipboard used to be set with Wails'
`runtime.ClipboardSetText` (plain text, never cleaned up), so dictated patient text stayed in Windows'
clipboard history (Win+V) / cloud clipboard. Now, on Windows only:

- `InsertTranscript`/`CopyToClipboard` call `setClipboardText` (`clipboard_windows.go` →
  `internal/winclip`), which additionally sets the registered formats
  `ExcludeClipboardContentFromMonitorProcessing`, `CanIncludeInClipboardHistory` (DWORD 0) and
  `CanUploadToCloudClipboard` (DWORD 0) (Microsoft Learn → "Clipboard Formats"). If marking fails the
  clipboard is emptied and the insert aborts (fail closed); if setting fails the insert aborts too, since
  Ctrl+V would otherwise paste stale foreign content.
- After the paste, `clientpolicy.Cleaner` empties the clipboard after `clipboard_clear_after_seconds`
  (default 30) **only if `GetClipboardSequenceNumber` is unchanged**, and again on shutdown
  (`OnShutdown` → `App.shutdown`). A typed insert (Citrix, `usesClipboardPath() == false`) doesn't touch the
  clipboard at all and must not cancel an earlier insert's pending clear (`Schedule` is only called when
  this insert set the clipboard).
- macOS/Linux keep the old behaviour (`clipboard_other.go`); no marking, no auto-clear.
- `GetInsertTarget()` (`insert_target.go`) reports the foreground program (`internal/winfocus`: exe name +
  window title); the overlay shows "Einfügen in: …". **Never log or persist the window title** — clinical
  software puts patient names in it. An optional allow-list (`insert_allowlist`, default empty = off) makes
  the frontend ask before inserting into an unlisted program (`acceptAndInsert` in `Recorder.tsx`, also for the
  hotkey/dictation-mic path). The client is a general-purpose dictation tool (e.g. private e-mails), so
  never make a list the default, and don't tie it to openDox categories — users can create their own.
- All three settings live in `settings.json` (see README); the logic that needs no OS/Wails is in
  `internal/clientpolicy` (unit-tested). Package `main` imports `hotkey`, which panics without an X11 display,
  so run main-package tests as `xvfb-run -a go test ./...`; the Windows-only `internal/` packages are checked
  with `GOOS=windows go vet ./internal/...` (the rest of Windows `main` can't be cross-compiled here because
  of robotgo/cgo). Behavior on real Windows (Win+V, cloud clipboard) still has to be checked by hand.

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
## Open-source hygiene and releases

The repo mirrors the server repo's governance setup (same shape as `~/python/project_red`):

- **Docs:** `README.md` (German, setup/config), `docs/opendox-client.md` (overview),
  `docs/data-mapping.md` (data flow / local storage / known gaps — update it whenever data handling,
  network targets, logging, clipboard or `localStorage` keys change), `docs/audit-checklist-opendox-client.md`
  (self-assessment with evidence), `SECURITY.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`,
  `CHANGELOG.md` (Keep a Changelog; add user-visible changes under `[Unreleased]`, German entries).
- **Version:** single source of truth is `info.productVersion` in `wails.json`. Never hand-edit it or
  `frontend/package.json`; run `python3 scripts/bump_version.py patch|minor|major [--commit]`, which also
  syncs the lockfile, moves `[Unreleased]` in the changelog and (with `--commit`) commits and tags `vX.Y.Z`.
  Pushing the tag runs `.github/workflows/release.yml` (checks tag == `wails.json`, publishes the changelog
  section as release notes). Installers are built locally (signed) and attached with `scripts/release.sh`
  under **fixed file names** — the server's admin UI links to `releases/latest/download/<name>`.
- **CI** (`.github/workflows/ci.yml`): gofmt (excl. `third_party/`), `go vet`, `xvfb-run go test`
  (with `-tags webkit2_41`; `frontend/dist` must be built first because of `//go:embed`),
  `GOOS=windows go vet ./internal/...`, frontend typecheck + build, REUSE lint. Dependabot covers
  gomod, npm and Actions. Not verified on GitHub until the first push.
- **REUSE / SPDX:** every `.go`/`.ts(x)`/`.py` source file carries the AGPL header plus the SPDX short tags
  (`python3 scripts/add_spdx_headers.py` adds the tags to headered files); everything else is covered by
  `REUSE.toml`, including vendored `third_party/` (licenses taken from the upstream notices) and the Wails
  template files. Check with `pip install reuse && reuse lint`. A new file without a header must either get one
  or an explicit entry in `REUSE.toml`.
- **Dependencies:** `python3 scripts/check_licenses.py [--write-notices]` checks Go (union of linux/windows/darwin
  `go list -deps`) and production npm licenses against an AGPL-compatible allowlist and regenerates
  `THIRD_PARTY_NOTICES.md` — never hand-edit that file.
- `build/windows/installer/tmp/` (Microsoft WebView2 bootstrapper) is gitignored: Wails downloads it for the
  NSIS build; don't commit proprietary binaries.

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

