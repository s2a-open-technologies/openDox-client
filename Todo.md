# Todo

## Diktiermikro-Tasten (Philips SpeechMike / Nuance PowerMic)

**Status:** Umgesetzt (Scaffold), unverifiziert — Byte-Offsets/Endianness sind aus der Referenz
übernommen, aber noch nicht gegen echte Hardware geprüft. `ODX_DICTATION_MIC_DEBUG=1` loggt rohe
Input-Reports zur Verifikation, sobald ein Gerät verfügbar ist (siehe `dictation_mic.go`,
`docs/superpowers/specs/2026-08-06-dictation-mic-design.md`).

Ziel: Physische Tasten (Aufnahme, Stopp, Rücklauf, Insert/Overwrite) an SpeechMike/PowerMic sollen
dieselben Aktionen wie die bestehenden globalen Hotkeys auslösen. Einsatz laut Planung an
Windows-Maschinen.

- Geräte senden Tasten über herstellerspezifisches USB-HID-Protokoll (nicht als normale
  Tastatureingaben) — der bestehende `golang.design/x/hotkey`-Mechanismus in `app.go` reicht dafür
  nicht.
- Referenzprotokoll (VID/PID, Report-Aufbau, Button-Bitmasken) ist bei
  `GoogleChromeLabs/dictation_support` (Apache-2.0) vollständig dokumentiert — spart uns
  Reverse-Engineering, wird nach Go portiert statt neu erschlossen.
- Geplante Umsetzung: neue Datei `dictation_mic.go` (Muster wie `hotkey_mod_*.go`), HID-Library
  `github.com/karalabe/hid`, Events per `runtime.EventsEmit` ans Frontend — analog zum bestehenden
  Hotkey-Muster in `Recorder.tsx` (`EventsOn`).
- Keine echte Datenbank nötig, nur eine fest kompilierte VID/PID- und Bitmasken-Tabelle.
- Windows braucht keine Sonderberechtigung (treiberloser HID-Zugriff). Linux/macOS sind für dieses
  Feature nicht der Einsatzfall, laufen aber aktiv mit (kein Build-Tag auf `dictation_mic.go`).

Vollständige Architektur-Notizen: `~/.claude/plans/wir-wurden-unterbrochen-der-abstract-umbrella.md`
(lokal bei Claude, nicht Teil des Repos).
