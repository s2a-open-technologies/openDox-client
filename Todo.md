# Todo

## Diktiermikro-Tasten (Philips SpeechMike / Nuance PowerMic)

**Status:** Architektur geklärt, Umsetzung ausstehend — wartet auf physische Hardware zum Testen.

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
  Feature nicht der Einsatzfall, bleiben aber build-fähig.
- Vor der Umsetzung: rohe Input-Reports mit echtem Gerät gegen die Referenz verifizieren, bevor
  Buttons scharf geschaltet werden (Byte-Offsets/Endianness ungetestet ohne Hardware).

Vollständige Architektur-Notizen: `~/.claude/plans/wir-wurden-unterbrochen-der-abstract-umbrella.md`
(lokal bei Claude, nicht Teil des Repos).
