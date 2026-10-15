// openDox - Medical transcription platform
// Copyright (C) 2026  S2A Open Technologies gGmbH
// SPDX-FileCopyrightText: 2026 S2A Open Technologies gGmbH
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ricknicius99/hid"
)

// dictationMicDebounce verhindert, dass ein einzelner Tastendruck durch mehrere Input-Reports
// (z.B. Press- und separater Bitmask-Update-Report) mehrfach ein Event auslöst. Gleicher Wert wie
// das Debounce für die Tastatur-Hotkeys in app.go.
const dictationMicDebounce = 700 * time.Millisecond

// dictationMicRescanInterval steuert, wie oft nach neu angeschlossenen Geräten gesucht wird
// (kein Hotplug-Callback in karalabe/hid verfügbar).
const dictationMicRescanInterval = 5 * time.Second

// dictationMicWorker serialisiert jeden Aufruf ins hid-Paket (Enumerate und Open) auf einen
// einzigen, dauerhaft fixierten OS-Thread. Grund: Das macOS-Backend (IOHIDManager) wird beim
// ersten Aufruf lazy erzeugt und an den CFRunLoop GENAU DIESES Threads gebunden. Go-Goroutinen
// sind aber nicht an OS-Threads gebunden — ein `go func(){ hid.Enumerate(...) }()` pro Scan (wie
// hier früher) landet bei jedem Aufruf potenziell auf einem anderen OS-Thread. Das ist ein
// Thread-Affinitäts-Verstoß gegen IOKits Mach-Port-/Run-Loop-Buchführung, den ältere macOS-
// Versionen stillschweigend toleriert haben; macOS 26s gehärtete Mach-Port-Guards beenden den
// Prozess dafür jetzt fatal (EXC_GUARD/GUARD_TYPE_MACH_PORT, "INVALID_OPTIONS on mach port 0") —
// reproduzierbar beim zweiten Scan (erster Aufruf initialisiert hid_mgr auf Thread A, der
// nächste Scan landet in einer frischen Goroutine auf Thread B).
type dictationMicWorker struct {
	jobs chan func()
}

func newDictationMicWorker() *dictationMicWorker {
	w := &dictationMicWorker{jobs: make(chan func())}
	go func() {
		runtime.LockOSThread()
		for job := range w.jobs {
			job()
		}
	}()
	return w
}

// run führt fn auf dem fixierten Worker-Thread aus und wartet auf Fertigstellung, gibt aber nach
// timeout auf (z.B. wenn der HID-Manager bei einem nicht reagierenden Gerät blockiert).
func (w *dictationMicWorker) run(timeout time.Duration, fn func()) bool {
	done := make(chan struct{})
	select {
	case w.jobs <- func() { fn(); close(done) }:
	case <-time.After(timeout):
		return false
	}
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

// startDictationMic sucht beim Start und danach periodisch nach unterstützten Diktiermikros
// (SpeechMike/PowerMic, siehe dictation_mic_protocol.go) und liest deren Tasten-Reports. Jeder
// Fehler (kein Gerät, kein libusb, Berechtigung fehlt) wird nur geloggt — die App inkl.
// Tastatur-Hotkeys muss unabhängig davon voll funktionieren.
func (a *App) startDictationMic() {
	// Not-Aus: falls ein (noch ungetestetes) Diktiermikro Fehl-Events feuert —
	// siehe Design-Doc "Bekanntes Risiko: Report-ID-Offset" — lässt sich der
	// komplette Mic-Pfad hiermit abschalten, ohne die Tastatur-Hotkeys zu berühren.
	if os.Getenv("ODX_DICTATION_MIC_DISABLE") == "1" {
		log.Printf("[opendox] dictation mic: deaktiviert (ODX_DICTATION_MIC_DISABLE=1)")
		return
	}
	if !hid.Supported() {
		log.Printf("[opendox] dictation mic: HID nicht unterstützt auf dieser Plattform")
		return
	}

	tracker := &dictationMicTracker{opened: make(map[string]bool)}
	worker := newDictationMicWorker()

	a.scanDictationMicsWithTracker(worker, tracker)

	go func() {
		ticker := time.NewTicker(dictationMicRescanInterval)
		defer ticker.Stop()
		for range ticker.C {
			a.scanDictationMicsWithTracker(worker, tracker)
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

func (a *App) scanDictationMicsWithTracker(worker *dictationMicWorker, tracker *dictationMicTracker) {
	results := scanDictationMicDevices(5 * time.Second)
	for _, r := range results {
		profile := matchDictationMicProfile(r.VendorID, r.ProductID)
		if profile == nil {
			continue // sollte nicht vorkommen, der Scan-Helper filtert schon
		}
		if !tracker.markOpened(r.Path) {
			continue // schon offen
		}
		info := hid.DeviceInfo{Path: r.Path, VendorID: r.VendorID, ProductID: r.ProductID}
		var device *hid.Device
		var err error
		if ok := worker.run(5*time.Second, func() {
			device, err = info.Open()
		}); !ok {
			log.Printf("[opendox] dictation mic: öffnen von %s (%s) nach 5s abgebrochen", info.Path, profile.name)
			tracker.markClosed(info.Path)
			continue
		}
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
func (a *App) readDictationMicLoop(info hid.DeviceInfo, device *hid.Device, profile *micProfile, tracker *dictationMicTracker) {
	var lastMask micButtonMask
	// recordHeld: Record-Taste wurde im Hold-Modus gedrückt und ist noch nicht losgelassen.
	// Der Modus wird pro Tastendruck gelesen (die Server-Richtlinie kann ihn zur Laufzeit
	// ändern); das Loslassen gehört immer zu dem Modus, in dem gedrückt wurde.
	recordHeld := false
	defer func() {
		// Gerät weg, während die Record-Taste gehalten wurde: sonst läuft die Aufnahme ewig.
		if recordHeld {
			a.emitMicRelease()
		}
		_ = device.Close()
		tracker.markClosed(info.Path)
		log.Printf("[opendox] dictation mic: getrennt - %s (pfad=%s)", profile.name, info.Path)
	}()

	debug := os.Getenv("ODX_DICTATION_MIC_DEBUG") == "1"
	buf := make([]byte, 64)
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
		falling := fallingEdge(lastMask, mask)
		lastMask = mask
		// Loslassen nie durch den Debounce verwerfen, sonst hängt die Aufnahme.
		if falling&micRecord != 0 && recordHeld {
			recordHeld = false
			a.emitMicRelease()
		}
		if rising&micRecord != 0 && a.currentPolicy().MicHoldMode() {
			recordHeld = true
			a.emitHotkeyEvent(eventMicRecordDown)
			lastEmit = time.Now()
			rising &^= micRecord
		}
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
// Im Hold-Modus hat readDictationMicLoop das Record-Bit schon herausgenommen (down/up);
// Stop bleibt in beiden Modi ein Toggle (bricht eine laufende Aufnahme ab bzw. beendet sie).
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
