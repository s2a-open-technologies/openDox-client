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
	infos := hid.Enumerate(0, 0)
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
func (a *App) readDictationMicLoop(info hid.DeviceInfo, device *hid.Device, profile *micProfile, tracker *dictationMicTracker) {
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
