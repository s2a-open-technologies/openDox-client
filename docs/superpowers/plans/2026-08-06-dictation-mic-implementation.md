# Dictation Mic Button Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Physical buttons on Philips SpeechMike / Nuance PowerMic devices (Record, Stop, Rewind,
Insert/Overwrite) fire the same Wails events as the existing global keyboard hotkeys
(`hotkey:toggle-record`, `hotkey:toggle-pause`, `hotkey:accept-insert`), requiring zero frontend
changes.

**Architecture:** A new pure protocol layer (`dictation_mic_protocol.go`) ports the VID/PID table
and report-parsing logic from `GoogleChromeLabs/dictation_support` (Apache-2.0) into small,
fully-unit-tested functions with no hardware dependency. A separate I/O layer
(`dictation_mic.go`) uses `github.com/karalabe/hid` to enumerate/open matching devices, read
input reports in per-device goroutines, and emit the same hotkey events `app.go` already defines
— through a shared helper that also respects the existing overlay-visibility gate.

**Tech Stack:** Go 1.26 (module targets go 1.25.0), `github.com/karalabe/hid` (BSD-3-Clause,
cgo/libusb-backed on Linux), stdlib `encoding/binary` + `testing`.

## Global Constraints

- No hardware available to test against — byte offsets/bit positions are ported verbatim from the
  reference implementation's actual source (verified, not guessed) but are **unverified against a
  real device**. Do not add any "just in case" fallback/guessing logic for offsets — ship exactly
  what's documented in `docs/superpowers/specs/2026-08-06-dictation-mic-design.md`, and gate raw
  verification behind the `ODX_DICTATION_MIC_DEBUG=1` env var.
- Runs on all platforms (no `//go:build` tag on the new files) — confirmed with the user.
- Every HID error (enumerate/open/read, missing `libusb` at runtime) must be non-fatal
  (`log.Printf` only) — the app including keyboard hotkeys must work with zero mic hardware
  attached and even if `libusb` isn't installed.
- Reuse the exact existing event strings `"hotkey:toggle-record"`, `"hotkey:toggle-pause"`,
  `"hotkey:accept-insert"` — do not invent new event names, since the goal is zero frontend
  changes.
- Mic-button events must respect the same `a.visible` overlay gate that keyboard hotkeys already
  respect (`app.go`'s `registerHotkey`) — via one shared helper, not duplicated logic.
- No test framework exists in this repo (`AGENTS.md`: "don't assume npm test, go test, or a
  linter") — but plain `go test ./...` needs zero setup for a Go module, so pure-logic functions
  in this feature ARE unit-tested with stdlib `testing`. HID I/O itself is not automatable without
  hardware; its "test" is a successful `go build` plus manual smoke-check that the app still
  starts and keyboard hotkeys still work with no device attached.
- This machine is missing the `libusb-1.0-0-dev` system package (`pkg-config --exists libusb-1.0`
  currently fails) — Task 1 installs it. Without it, `go build` for this feature will fail with a
  cgo/pkg-config error.

---

### Task 1: Add `karalabe/hid` dependency and Linux build prerequisite

**Files:**
- Modify: `go.mod`, `go.sum`
- Modify: `.agents/AGENTS.md:33-37` (Linux build dependency section)

**Interfaces:**
- Produces: `github.com/karalabe/hid` importable as `hid` in later tasks (`hid.Supported()`,
  `hid.Enumerate(vendorID, productID uint16) ([]hid.DeviceInfo, error)`,
  `hid.DeviceInfo.Open() (hid.Device, error)`, `hid.Device.Read(b []byte) (int, error)`,
  `hid.Device.Close() error`).

- [ ] **Step 1: Install the missing Linux build dependency**

Run: `sudo apt-get install -y libusb-1.0-0-dev`

Expected: package installs successfully. Verify with:

Run: `pkg-config --exists libusb-1.0 && echo OK`
Expected: `OK`

- [ ] **Step 2: Add the Go dependency**

Run: `go get github.com/karalabe/hid@latest`

Expected: `go.mod` gains a `require github.com/karalabe/hid vX.Y.Z` line, `go.sum` gains matching
entries.

- [ ] **Step 3: Verify it compiles with a throwaway import check**

Run: `go build ./...`

Expected: succeeds with no changes to any `.go` file yet (this only proves the dependency resolves
and the existing build is unaffected before we write any code against it).

- [ ] **Step 4: Document the new Linux build dependency**

In `.agents/AGENTS.md`, find the existing section:

```markdown
## Linux build dependency

`webview_perms.go` uses cgo against `webkit2gtk-4.1` and `gtk+-3.0` (via `pkg-config`) with **no Go build
tag**, so it compiles unconditionally. In practice this only builds on Linux even though `build/darwin`
and `build/windows` assets exist — building on macOS/Windows fails unless this file gets a
`//go:build linux` constraint (and a no-op stub for other platforms).
```

Replace it with:

```markdown
## Linux build dependency

`webview_perms.go` uses cgo against `webkit2gtk-4.1` and `gtk+-3.0` (via `pkg-config`) with **no Go build
tag**, so it compiles unconditionally. In practice this only builds on Linux even though `build/darwin`
and `build/windows` assets exist — building on macOS/Windows fails unless this file gets a
`//go:build linux` constraint (and a no-op stub for other platforms).

`dictation_mic.go` uses `github.com/karalabe/hid`, which on Linux is backed by `libusb` via cgo —
needs `libusb-1.0-0-dev` (`pkg-config libusb-1.0`) installed to build. Windows/macOS builds of this
dependency don't need an extra system package.
```

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum .agents/AGENTS.md
git commit -m "Add karalabe/hid dependency for dictation mic support"
```

---

### Task 2: Extract shared hotkey-event constants and visibility-gated emit helper in `app.go`

Pure refactor of existing code — no behavior change — so both the keyboard hotkeys and (in Task 4)
the mic buttons emit through one path.

**Files:**
- Modify: `app.go:82-120` (`registerGlobalHotkeys`, `registerHotkey`)

**Interfaces:**
- Produces: constants `eventToggleRecord`, `eventTogglePause`, `eventAcceptInsert` (all `string`);
  method `func (a *App) emitHotkeyEvent(eventName string)`.
- Consumes: existing `a.mu`, `a.visible` fields (already defined on `App` in `app.go:63-69`).

- [ ] **Step 1: Replace the hardcoded event strings with named constants and a shared emit helper**

In `app.go`, replace:

```go
// registerGlobalHotkeys registriert die systemweiten Hotkeys (funktionieren unabhängig
// davon, welches Fenster gerade fokussiert ist) und meldet jeden Tastendruck als
// Wails-Event an das Frontend.
func (a *App) registerGlobalHotkeys() {
	// hotkeyMods ist plattformspezifisch definiert (hotkey_mod_*.go) — sowohl die
	// Modifier-Konstanten als auch die übliche Tastenkombination (Ctrl+Alt vs.
	// Cmd+Option auf macOS) unterscheiden sich pro OS.
	mods := hotkeyMods
	a.registerHotkey(mods, hotkey.KeyR, "hotkey:toggle-record")
	a.registerHotkey(mods, hotkey.KeyP, "hotkey:toggle-pause")
	a.registerHotkey(mods, hotkey.KeyV, "hotkey:accept-insert")
	a.registerWindowHotkey(mods, hotkey.KeyH) // Ctrl+Alt+H: Overlay ein-/ausblenden
}

// registerHotkey registriert eine einzelne Tastenkombination und emittiert bei jedem
// (entprellten) Tastendruck das angegebene Wails-Event.
func (a *App) registerHotkey(mods []hotkey.Modifier, key hotkey.Key, eventName string) {
	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		log.Printf("[opendox] failed to register global hotkey for %s: %v", eventName, err)
		return
	}
	go func() {
		var last time.Time
		for range hk.Keydown() {
			if time.Since(last) < 700*time.Millisecond {
				continue // debounce X11 auto-repeat while die Kombination gehalten wird
			}
			last = time.Now()
			a.mu.Lock()
			visible := a.visible
			a.mu.Unlock()
			if !visible {
				continue // Overlay ausgeblendet -> Ctrl+Alt+R / Ctrl+Alt+P / Ctrl+Alt+V deaktiviert
			}
			runtime.EventsEmit(a.ctx, eventName)
		}
	}()
}
```

with:

```go
// Event-Namen, die sowohl von den Tastatur-Hotkeys als auch von den physischen
// Diktiermikro-Tasten (dictation_mic.go) emittiert werden — Frontend kennt nur diese drei.
const (
	eventToggleRecord = "hotkey:toggle-record"
	eventTogglePause  = "hotkey:toggle-pause"
	eventAcceptInsert = "hotkey:accept-insert"
)

// registerGlobalHotkeys registriert die systemweiten Hotkeys (funktionieren unabhängig
// davon, welches Fenster gerade fokussiert ist) und meldet jeden Tastendruck als
// Wails-Event an das Frontend.
func (a *App) registerGlobalHotkeys() {
	// hotkeyMods ist plattformspezifisch definiert (hotkey_mod_*.go) — sowohl die
	// Modifier-Konstanten als auch die übliche Tastenkombination (Ctrl+Alt vs.
	// Cmd+Option auf macOS) unterscheiden sich pro OS.
	mods := hotkeyMods
	a.registerHotkey(mods, hotkey.KeyR, eventToggleRecord)
	a.registerHotkey(mods, hotkey.KeyP, eventTogglePause)
	a.registerHotkey(mods, hotkey.KeyV, eventAcceptInsert)
	a.registerWindowHotkey(mods, hotkey.KeyH) // Ctrl+Alt+H: Overlay ein-/ausblenden
}

// registerHotkey registriert eine einzelne Tastenkombination und emittiert bei jedem
// (entprellten) Tastendruck das angegebene Wails-Event.
func (a *App) registerHotkey(mods []hotkey.Modifier, key hotkey.Key, eventName string) {
	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		log.Printf("[opendox] failed to register global hotkey for %s: %v", eventName, err)
		return
	}
	go func() {
		var last time.Time
		for range hk.Keydown() {
			if time.Since(last) < 700*time.Millisecond {
				continue // debounce X11 auto-repeat while die Kombination gehalten wird
			}
			last = time.Now()
			a.emitHotkeyEvent(eventName)
		}
	}()
}

// emitHotkeyEvent meldet eines der drei Hotkey-Events ans Frontend, sofern das Overlay nicht
// per Ctrl+Alt+H ausgeblendet ist. Gemeinsamer Pfad für Tastatur-Hotkeys (oben) und physische
// Diktiermikro-Tasten (dictation_mic.go) — beide sollen sich bei ausgeblendetem Overlay gleich
// verhalten.
func (a *App) emitHotkeyEvent(eventName string) {
	a.mu.Lock()
	visible := a.visible
	a.mu.Unlock()
	if !visible {
		return
	}
	runtime.EventsEmit(a.ctx, eventName)
}
```

- [ ] **Step 2: Verify it builds**

Run: `go build -tags dev,webkit2_41 ./...`

Expected: succeeds with no errors.

- [ ] **Step 3: Manual smoke check (no automated test possible for hotkey/window-manager interaction)**

Run: `wails dev` (or `go run -tags dev,webkit2_41 .` with `cd frontend && npm run dev` running
separately), then press `Ctrl+Alt+R`. Expected: recording starts/stops exactly as before this
refactor — behavior must be unchanged, this step only proves the extraction didn't regress it.

- [ ] **Step 4: Commit**

```bash
git add app.go
git commit -m "Extract shared hotkey event constants and emit helper in app.go"
```

---

### Task 3: Pure protocol layer — device matching and report parsing (fully unit-tested)

**Files:**
- Create: `dictation_mic_protocol.go`
- Create: `dictation_mic_protocol_test.go`

**Interfaces:**
- Consumes: nothing (stdlib only — `encoding/binary`).
- Produces: type `micButtonMask uint16` with constants `micRecord`, `micStop`, `micRewind`,
  `micInsOvr`; type `micProfile struct { name string; parse func([]byte) (micButtonMask, bool) }`;
  function `matchDictationMicProfile(vendorID, productID uint16) *micProfile`; function
  `risingEdge(prev, cur micButtonMask) micButtonMask`. Task 4 imports all of these.

- [ ] **Step 1: Write the failing tests**

Create `dictation_mic_protocol_test.go`:

```go
package main

import "testing"

func TestMatchDictationMicProfile(t *testing.T) {
	cases := []struct {
		name                string
		vendorID, productID uint16
		wantProfile         string // "" means no match expected
	}{
		{"SpeechMike wired LFH/SMP", 0x0911, 0x0c1c, "SpeechMike/PowerMic IV"},
		{"SpeechMike Premium Air", 0x0911, 0x0c1d, "SpeechMike/PowerMic IV"},
		{"SpeechOne/Ambient", 0x0911, 0x0c1e, "SpeechMike/PowerMic IV"},
		{"SpeechMike Browser/Gamepad mode HID interface", 0x0911, 0x0fa0, "SpeechMike/PowerMic IV"},
		{"PowerMic IV", 0x0554, 0x0064, "SpeechMike/PowerMic IV"},
		{"PowerMic III", 0x0554, 0x1001, "PowerMic III"},
		{"unrelated vendor", 0x1234, 0x5678, ""},
		{"Philips vendor, unknown product", 0x0911, 0x9999, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			profile := matchDictationMicProfile(c.vendorID, c.productID)
			if c.wantProfile == "" {
				if profile != nil {
					t.Fatalf("expected no match, got profile %q", profile.name)
				}
				return
			}
			if profile == nil {
				t.Fatalf("expected profile %q, got no match", c.wantProfile)
			}
			if profile.name != c.wantProfile {
				t.Fatalf("expected profile %q, got %q", c.wantProfile, profile.name)
			}
		})
	}
}

func TestParseSpeechMikeReport(t *testing.T) {
	cases := []struct {
		name     string
		report   []byte
		wantOK   bool
		wantMask micButtonMask
	}{
		// Command byte 0x80 (BUTTON_PRESS_EVENT), bitmask = LE uint16 at offset 7.
		{"record pressed (bit8 -> 0x0100)", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x01}, true, micRecord},
		{"stop pressed (bit9 -> 0x0200)", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x02}, true, micStop},
		{"rewind pressed (bit12 -> 0x1000)", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x10}, true, micRewind},
		{"ins/ovr pressed (bit14 -> 0x4000)", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x40}, true, micInsOvr},
		{"record+stop both set", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x03}, true, micRecord | micStop},
		{"no buttons pressed", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x00}, true, 0},
		{"unrelated bit ignored (F1_A, bit1)", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x02, 0x00}, true, 0},
		{"wrong command byte (motion event 0x9e) ignored", []byte{0x9e, 0, 0, 0, 0, 0, 0, 0, 0}, false, 0},
		{"report too short", []byte{0x80, 0, 0}, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mask, ok := parseSpeechMikeReport(c.report)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if ok && mask != c.wantMask {
				t.Fatalf("mask = %v, want %v", mask, c.wantMask)
			}
		})
	}
}

func TestParsePowerMic3Report(t *testing.T) {
	cases := []struct {
		name     string
		report   []byte
		wantOK   bool
		wantMask micButtonMask
	}{
		// No command byte check, bitmask = LE uint16 at offset 1.
		{"record pressed (bit2 -> 0x0004)", []byte{0, 0x04, 0x00}, true, micRecord},
		{"rewind pressed (bit4 -> 0x0010)", []byte{0, 0x10, 0x00}, true, micRewind},
		{"no buttons pressed", []byte{0, 0x00, 0x00}, true, 0},
		{"unrelated bit ignored (TRANSCRIBE, bit0)", []byte{0, 0x01, 0x00}, true, 0},
		{"report too short", []byte{0, 0x04}, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mask, ok := parsePowerMic3Report(c.report)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if ok && mask != c.wantMask {
				t.Fatalf("mask = %v, want %v", mask, c.wantMask)
			}
		})
	}
}

func TestRisingEdge(t *testing.T) {
	cases := []struct {
		name string
		prev, cur, want micButtonMask
	}{
		{"press from idle", 0, micRecord, micRecord},
		{"held (no new edge)", micRecord, micRecord, 0},
		{"release (no edge)", micRecord, 0, 0},
		{"second button added while first held", micRecord, micRecord | micStop, micStop},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := risingEdge(c.prev, c.cur); got != c.want {
				t.Fatalf("risingEdge(%v, %v) = %v, want %v", c.prev, c.cur, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./... -run 'TestMatchDictationMicProfile|TestParseSpeechMikeReport|TestParsePowerMic3Report|TestRisingEdge' -v`

Expected: FAIL — compile error, since `matchDictationMicProfile`, `parseSpeechMikeReport`,
`parsePowerMic3Report`, `risingEdge`, `micButtonMask`, `micRecord`, `micStop`, `micRewind`,
`micInsOvr` don't exist yet.

- [ ] **Step 3: Write the implementation**

Create `dictation_mic_protocol.go`:

```go
package main

import "encoding/binary"

// micButtonMask ist unsere eigene, geräteunabhängige Bit-Repräsentation der vier Tasten, die wir
// verdrahten. Die Bit-Positionen hier sind intern und haben keine Bedeutung außerhalb dieses
// Programms — sie sind NICHT identisch mit den Roh-Bitmasken der Geräte (siehe parse*-Funktionen
// unten für die Übersetzung).
type micButtonMask uint16

const (
	micRecord micButtonMask = 1 << 0
	micStop   micButtonMask = 1 << 1
	micRewind micButtonMask = 1 << 2
	micInsOvr micButtonMask = 1 << 3
)

// micProfile beschreibt, wie ein Gerät identifiziert und dessen Input-Reports interpretiert
// werden. parse gibt ok=false zurück, wenn der Report ignoriert werden soll (z.B. falscher
// Command-Byte oder zu kurz).
type micProfile struct {
	name  string
	parse func(report []byte) (micButtonMask, bool)
}

// matchDictationMicProfile ordnet eine USB-VID/PID-Kombination einem bekannten Diktiermikro-Profil
// zu. VID/PID-Tabelle 1:1 aus GoogleChromeLabs/dictation_support (dictation_device_manager.ts,
// DEVICE_FILTERS) übernommen — siehe docs/superpowers/specs/2026-08-06-dictation-mic-design.md.
func matchDictationMicProfile(vendorID, productID uint16) *micProfile {
	if matchesSpeechMikeClass(vendorID, productID) {
		return &micProfile{name: "SpeechMike/PowerMic IV", parse: parseSpeechMikeReport}
	}
	if matchesPowerMic3(vendorID, productID) {
		return &micProfile{name: "PowerMic III", parse: parsePowerMic3Report}
	}
	return nil
}

func matchesSpeechMikeClass(vendorID, productID uint16) bool {
	if vendorID == 0x0911 {
		switch productID {
		case 0x0c1c, 0x0c1d, 0x0c1e, 0x0fa0:
			return true
		}
		return false
	}
	return vendorID == 0x0554 && productID == 0x0064 // PowerMic IV
}

func matchesPowerMic3(vendorID, productID uint16) bool {
	return vendorID == 0x0554 && productID == 0x1001
}

// parseSpeechMikeReport übersetzt einen Input-Report der SpeechMike-Geräteklasse (inkl. PowerMic
// IV) in unsere interne Bitmaske. Referenz: speechmike_hid_device.ts, onInputReport/
// getInputBitmask/BUTTON_MAPPINGS_SPEECHMIKE. Byte-Offsets sind gegen den WebHID-Referenzcode
// verifiziert, aber NICHT gegen echte Hardware über hidapi (siehe Design-Doc, Abschnitt
// "Bekanntes Risiko: Report-ID-Offset").
func parseSpeechMikeReport(report []byte) (micButtonMask, bool) {
	if len(report) < 9 {
		return 0, false
	}
	const buttonPressEvent = 0x80
	if report[0] != buttonPressEvent {
		return 0, false
	}
	raw := binary.LittleEndian.Uint16(report[7:9])
	var mask micButtonMask
	if raw&(1<<8) != 0 {
		mask |= micRecord
	}
	if raw&(1<<9) != 0 {
		mask |= micStop
	}
	if raw&(1<<12) != 0 {
		mask |= micRewind
	}
	if raw&(1<<14) != 0 {
		mask |= micInsOvr
	}
	return mask, true
}

// parsePowerMic3Report übersetzt einen Input-Report von PowerMic III in unsere interne Bitmaske.
// Referenz: powermic_3_device.ts, getInputBitmask/BUTTON_MAPPINGS. Kein Command-Byte-Check (die
// Referenz nutzt hier den Default-Pfad der Basisklasse). PowerMic III hat keine physische
// Stop-/Insert-Overwrite-Taste, daher tauchen micStop/micInsOvr hier nie auf.
func parsePowerMic3Report(report []byte) (micButtonMask, bool) {
	if len(report) < 3 {
		return 0, false
	}
	raw := binary.LittleEndian.Uint16(report[1:3])
	var mask micButtonMask
	if raw&(1<<2) != 0 {
		mask |= micRecord
	}
	if raw&(1<<4) != 0 {
		mask |= micRewind
	}
	return mask, true
}

// risingEdge liefert die Bits, die in cur neu gesetzt sind gegenüber prev (0->1-Übergang) — nur
// diese sollen eine Aktion auslösen, nicht jeder wiederholte Report mit gehaltener Taste.
func risingEdge(prev, cur micButtonMask) micButtonMask {
	return cur &^ prev
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./... -run 'TestMatchDictationMicProfile|TestParseSpeechMikeReport|TestParsePowerMic3Report|TestRisingEdge' -v`

Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add dictation_mic_protocol.go dictation_mic_protocol_test.go
git commit -m "Add pure dictation-mic protocol parsing with unit tests"
```

---

### Task 4: HID device I/O — enumerate, open, read loop, wire to hotkey events

**Files:**
- Create: `dictation_mic.go`
- Modify: `app.go:77-80` (`startup`)

**Interfaces:**
- Consumes: `matchDictationMicProfile`, `micProfile`, `micButtonMask`, `micRecord`, `micStop`,
  `micRewind`, `micInsOvr`, `risingEdge` (Task 3); `eventToggleRecord`, `eventTogglePause`,
  `eventAcceptInsert`, `a.emitHotkeyEvent` (Task 2); `hid.Supported`, `hid.Enumerate`,
  `hid.DeviceInfo`, `hid.Device` (Task 1).
- Produces: `func (a *App) startDictationMic()`, called once from `startup()`.

- [ ] **Step 1: Write the implementation**

Create `dictation_mic.go`:

```go
package main

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/karalabe/hid"
)

// dictationMicDebounce verhindert, dass ein einzelner Tastendruck durch mehrere Input-Reports
// (z.B. Press- und separater Bitmask-Update-Report) mehrfach ein Event auslöst. Gleicher Wert wie
// das Debounce für die Tastatur-Hotkeys in app.go.
const dictationMicDebounce = 700 * time.Millisecond

// dictationMicRescanInterval steuert, wie oft nach neu angeschlossenen Geräten gesucht wird
// (kein Hotplug-Callback in karalabe/hid verfügbar).
const dictationMicRescanInterval = 5 * time.Second

// startDictationMic sucht beim Start und danach periodisch nach unterstützten Diktiermikros
// (SpeechMike/PowerMic, siehe dictation_mic_protocol.go) und liest deren Tasten-Reports. Jeder
// Fehler (kein Gerät, kein libusb, Berechtigung fehlt) wird nur geloggt — die App inkl.
// Tastatur-Hotkeys muss unabhängig davon voll funktionieren.
func (a *App) startDictationMic() {
	if !hid.Supported() {
		log.Printf("[opendox] dictation mic: HID nicht unterstützt auf dieser Plattform")
		return
	}

	tracker := &dictationMicTracker{opened: make(map[string]bool)}

	a.scanDictationMics(tracker)
	go func() {
		ticker := time.NewTicker(dictationMicRescanInterval)
		defer ticker.Stop()
		for range ticker.C {
			a.scanDictationMics(tracker)
		}
	}()
}

// dictationMicTracker merkt sich, welche Geräte-Pfade bereits eine eigene Read-Goroutine haben,
// damit der periodische Rescan sie nicht doppelt öffnet.
type dictationMicTracker struct {
	mu     sync.Mutex
	opened map[string]bool
}

func (t *dictationMicTracker) markOpened(path string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.opened[path] {
		return false
	}
	t.opened[path] = true
	return true
}

func (t *dictationMicTracker) markClosed(path string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.opened, path)
}

func (a *App) scanDictationMics(tracker *dictationMicTracker) {
	infos, err := hid.Enumerate(0, 0)
	if err != nil {
		log.Printf("[opendox] dictation mic: enumerate fehlgeschlagen: %v", err)
		return
	}
	for _, info := range infos {
		profile := matchDictationMicProfile(info.VendorID, info.ProductID)
		if profile == nil {
			continue
		}
		if !tracker.markOpened(info.Path) {
			continue // schon offen
		}
		device, err := info.Open()
		if err != nil {
			log.Printf("[opendox] dictation mic: öffnen von %s (%s) fehlgeschlagen: %v", info.Path, profile.name, err)
			tracker.markClosed(info.Path)
			continue
		}
		log.Printf("[opendox] dictation mic: verbunden - %s (vid=%04x pid=%04x, pfad=%s)", profile.name, info.VendorID, info.ProductID, info.Path)
		go a.readDictationMicLoop(info, device, profile, tracker)
	}
}

// readDictationMicLoop liest blockierend Input-Reports von einem offenen Gerät, bis Read() einen
// Fehler liefert (z.B. Gerät abgezogen). Löst bei jeder steigenden Flanke einer verdrahteten Taste
// (nach Debounce) das zugehörige Hotkey-Event aus.
func (a *App) readDictationMicLoop(info hid.DeviceInfo, device hid.Device, profile *micProfile, tracker *dictationMicTracker) {
	defer func() {
		_ = device.Close()
		tracker.markClosed(info.Path)
		log.Printf("[opendox] dictation mic: getrennt - %s (pfad=%s)", profile.name, info.Path)
	}()

	debug := os.Getenv("ODX_DICTATION_MIC_DEBUG") == "1"
	buf := make([]byte, 64)
	var lastMask micButtonMask
	var lastEmit time.Time

	for {
		n, err := device.Read(buf)
		if err != nil {
			log.Printf("[opendox] dictation mic: Lesefehler auf %s (%s): %v", info.Path, profile.name, err)
			return
		}
		report := buf[:n]
		if debug {
			log.Printf("[opendox] dictation mic debug %s: % x", info.Path, report)
		}

		mask, ok := profile.parse(report)
		if !ok {
			continue
		}
		rising := risingEdge(lastMask, mask)
		lastMask = mask
		if rising == 0 {
			continue
		}
		if time.Since(lastEmit) < dictationMicDebounce {
			continue
		}
		lastEmit = time.Now()
		a.emitDictationMicButtons(rising)
	}
}

// emitDictationMicButtons setzt physische Tasten auf dieselben Hotkey-Events um wie die
// Tastatur-Hotkeys - siehe docs/superpowers/specs/2026-08-06-dictation-mic-design.md,
// Abschnitt "Mapping auf bestehende Hotkey-Events".
func (a *App) emitDictationMicButtons(rising micButtonMask) {
	if rising&(micRecord|micStop) != 0 {
		a.emitHotkeyEvent(eventToggleRecord)
	}
	if rising&micRewind != 0 {
		a.emitHotkeyEvent(eventTogglePause)
	}
	if rising&micInsOvr != 0 {
		a.emitHotkeyEvent(eventAcceptInsert)
	}
}
```

- [ ] **Step 2: Wire it into `startup()`**

In `app.go`, replace:

```go
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.registerGlobalHotkeys()
}
```

with:

```go
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.registerGlobalHotkeys()
	a.startDictationMic()
}
```

- [ ] **Step 3: Verify it builds**

Run: `go build -tags dev,webkit2_41 ./...`

Expected: succeeds. If it fails with a `pkg-config`/cgo error mentioning `libusb`, Task 1 Step 1
wasn't completed on this machine — stop and re-run it.

- [ ] **Step 4: Run the full test suite to confirm Task 3's tests still pass unaffected**

Run: `go test ./... -v`

Expected: all `TestMatchDictationMicProfile`, `TestParseSpeechMikeReport`,
`TestParsePowerMic3Report`, `TestRisingEdge` subtests still PASS (this task adds no new automated
tests — HID I/O itself can't be tested without hardware, see Global Constraints).

- [ ] **Step 5: Manual smoke check — app must work fine with zero mic hardware attached**

Run: `wails dev` (or `go run -tags dev,webkit2_41 .`, frontend dev server running separately).

Expected:
- App starts normally, no crash.
- Console/terminal log shows either nothing dictation-mic related, or only
  `[opendox] dictation mic: enumerate fehlgeschlagen: ...` / no matching devices — never a panic.
- `Ctrl+Alt+R` / `Ctrl+Alt+P` / `Ctrl+Alt+V` / `Ctrl+Alt+H` all still behave exactly as before.

- [ ] **Step 6: Commit**

```bash
git add dictation_mic.go app.go
git commit -m "Wire physical SpeechMike/PowerMic buttons to existing hotkey events"
```

---

### Task 5: Documentation — verification tooling, Todo.md status update

**Files:**
- Modify: `.agents/AGENTS.md` (new subsection under "Global hotkeys" or its own subsection)
- Modify: `Todo.md`

**Interfaces:**
- Consumes: nothing new — this task only documents Tasks 1-4's output.

- [ ] **Step 1: Document the feature and the debug env var in `.agents/AGENTS.md`**

Find the "### Global hotkeys (cross-boundary wiring)" section in `.agents/AGENTS.md` and add a new
subsection immediately after it:

```markdown
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
```

- [ ] **Step 2: Update `Todo.md` status**

Replace the `**Status:**` line and the "Vor der Umsetzung" bullet in `Todo.md`:

```markdown
**Status:** Umgesetzt (Scaffold), unverifiziert — Byte-Offsets/Endianness sind aus der Referenz
übernommen, aber noch nicht gegen echte Hardware geprüft. `ODX_DICTATION_MIC_DEBUG=1` loggt rohe
Input-Reports zur Verifikation, sobald ein Gerät verfügbar ist (siehe `dictation_mic.go`,
`docs/superpowers/specs/2026-08-06-dictation-mic-design.md`).
```

and remove the now-stale bullet:

```markdown
- Vor der Umsetzung: rohe Input-Reports mit echtem Gerät gegen die Referenz verifizieren, bevor
  Buttons scharf geschaltet werden (Byte-Offsets/Endianness ungetestet ohne Hardware).
```

(replace it with nothing — the `ODX_DICTATION_MIC_DEBUG` env var mentioned in the new Status line
now covers this).

- [ ] **Step 3: Commit**

```bash
git add .agents/AGENTS.md Todo.md
git commit -m "Document dictation mic feature and verification env var"
```

---

## Self-Review Notes

- **Spec coverage:** VID/PID table (Task 3), report parsing + byte offsets (Task 3), event mapping
  (Task 4), overlay-visibility gate (Task 2/4), debug/verification tooling (Task 4/5), new
  dependency + Linux build doc (Task 1), Todo.md status update (Task 5), cross-platform "runs
  everywhere" (no build tags anywhere in Tasks 3-4) — all covered.
- **No placeholders:** every step has literal code or exact file content, not descriptions.
- **Type consistency checked:** `micButtonMask` / `micRecord` / `micStop` / `micRewind` /
  `micInsOvr` / `micProfile` / `matchDictationMicProfile` / `risingEdge` are defined once in Task 3
  and used with identical names/signatures in Task 4's tests-passing check and implementation.
  `eventToggleRecord` / `eventTogglePause` / `eventAcceptInsert` / `emitHotkeyEvent` defined once in
  Task 2, consumed identically in Task 4.
