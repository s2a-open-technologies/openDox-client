# Diktiermikro-Tasten (Philips SpeechMike / Nuance PowerMic) — Design

**Status:** Design abgenommen, Implementierung ausstehend. Umsetzung erfolgt ohne physische
Hardware zum Testen — Byte-Offsets/Endianness sind aus der Referenzimplementierung übernommen,
aber unverifiziert (siehe Testing/Verifikation unten).

## Ziel

Physische Tasten an Philips SpeechMike (wired/Premium/SpeechOne/Ambient) und Nuance PowerMic
(III/IV) sollen dieselben Aktionen auslösen wie die bestehenden globalen Tastatur-Hotkeys
(`Ctrl+Alt+R/P/A` in `app.go`). Einsatz laut Planung primär auf Windows, aber es läuft auf allen
Plattformen (Linux/macOS eingeschlossen), auf denen `karalabe/hid` unterstützt wird.

## Referenz

Protokoll (VID/PID, Report-Layout, Button-Bitmasken) ist 1:1 aus
`GoogleChromeLabs/dictation_support` (Apache-2.0) übernommen — verifiziert gegen den tatsächlichen
Quellcode der Dateien `dictation_device_base.ts`, `speechmike_hid_device.ts`,
`powermic_3_device.ts`, `dictation_device_manager.ts` (nicht aus dem Gedächtnis rekonstruiert).

### Geräteerkennung (VID/PID)

Beide Gerätegruppen teilen sich eine Report-Struktur ("SpeechMike-Klasse"), PowerMic III hat eine
eigene:

| Gruppe | VID | PID | Geräte |
|---|---|---|---|
| SpeechMike-Klasse | `0x0911` | `0x0c1c` | Wired SpeechMike LFH35xx/36xx, SMP37xx/38xx |
| SpeechMike-Klasse | `0x0911` | `0x0c1d` | SpeechMike Premium Air SMP40xx |
| SpeechMike-Klasse | `0x0911` | `0x0c1e` | SpeechOne PSM6000, SpeechMike Ambient PSM5000 |
| SpeechMike-Klasse | `0x0911` | `0x0fa0` | Alle SpeechMikes im Browser/Gamepad-Modus (HID-Interface) |
| SpeechMike-Klasse | `0x0554` | `0x0064` | PowerMic IV |
| PowerMic III | `0x0554` | `0x1001` | PowerMic III |

Usage-Page/Usage-Filterung aus der Referenz (dient dort v.a. der Unterscheidung mehrerer
HID-Collections desselben Geräts) wird bewusst **nicht** übernommen — `karalabe/hid` liefert diese
Felder nur auf Windows/macOS zuverlässig, auf Linux (libusb-Backend) nicht garantiert. Matching
läuft nur über VID/PID. Risiko: falls ein Gerät mehrere HID-Interfaces mit identischer VID/PID
exponiert, könnten wir das falsche/zusätzliche Interface öffnen. Wird bei der
Hardware-Verifikation mitgeprüft.

### Report-Parsing

**SpeechMike-Klasse:**
- Report-Byte 0 muss `0x80` sein (Command `BUTTON_PRESS_EVENT`; andere Commands wie
  `MOTION_EVENT`/`WIRELESS_STATUS_EVENT` werden ignoriert).
- Bitmaske = little-endian `uint16` ab Byte-Offset 7.
- Verwendete Bits (aus `BUTTON_MAPPINGS_SPEECHMIKE`/`BUTTON_MAPPINGS_POWERMIC_4`, beide identisch
  für die hier genutzten vier Buttons): `RECORD = 1<<8`, `STOP = 1<<9`, `REWIND = 1<<12`,
  `INS_OVR = 1<<14`.

**PowerMic III:**
- Keine Command-Byte-Prüfung (Referenz nutzt hier den Default-Pfad der Basisklasse).
- Bitmaske = little-endian `uint16` ab Byte-Offset 1.
- Verwendete Bits (aus `BUTTON_MAPPINGS`): `RECORD = 1<<2`, `REWIND = 1<<4`. PowerMic III hat
  keine physische Stop- oder Insert/Overwrite-Taste — diese Bits existieren für dieses Gerät
  nicht und lösen dort nie aus. Das ist kein Fehlerfall, sondern erwartetes Verhalten.

### Bekanntes Risiko: Report-ID-Offset

Die Referenz liest über die WebHID-API, die die Report-ID (falls das Gerät "numbered reports"
nutzt) bereits aus `event.data` entfernt hat. `karalabe/hid` (hidapi-Backend) liefert je nach
Plattform ggf. die Report-ID als erstes Byte mit. Die oben genannten Offsets sind 1:1 aus der
WebHID-Referenz übernommen und **potenziell um 1 Byte verschoben** gegenüber dem, was
`hid.Device.Read()` tatsächlich liefert. Das ist exakt der Punkt, den das Todo als "vor der
Umsetzung verifizieren" markiert hat — siehe Verifikations-Tooling unten.

## Architektur

Neue Datei `dictation_mic.go` (Package `main`, kein Build-Tag — läuft auf allen Plattformen, wie
entschieden).

```
app.go:startup()
  └─ a.registerGlobalHotkeys()      (bestehend)
  └─ a.startDictationMic()          (neu)
       └─ hid.Supported() Guard
       └─ scanAndOpenDictationMics() — hid.Enumerate(0,0), Filter gegen Profil-Tabelle,
          pro neu gefundenem Pfad: Open() + eigene Goroutine mit Blocking-Read-Loop
       └─ periodischer Rescan-Ticker (z.B. alle 5s) für neu angeschlossene Geräte
```

Pro offenem Gerät:
- Blocking-Read-Loop, parst Report laut Profil (SpeechMike-Klasse oder PowerMic III) in eine
  `uint32`-Bitmaske der vier verwendeten Buttons.
- Hält `lastBits` je Gerät; nur bei steigender Flanke (0→1) eines Bits + 700ms-Debounce (identisches
  Muster wie `registerHotkey` in `app.go`) wird das zugehörige Frontend-Event gefeuert.
- Bei Read-Fehler (Gerät abgezogen): Goroutine beendet sich, Gerät wird aus der Tracking-Map
  entfernt und kann vom Rescan-Ticker erneut geöffnet werden, falls es wieder angeschlossen wird.
- Alle Fehler (Enumerate/Open/Read) werden nur per `log.Printf` gemeldet, niemals fatal — ohne
  Hardware oder ohne `libusb` zur Laufzeit muss die App inklusive Tastatur-Hotkeys unverändert
  funktionieren.

### Mapping auf bestehende Hotkey-Events

Die drei Event-Strings `"hotkey:toggle-record"`, `"hotkey:toggle-pause"`, `"hotkey:accept-insert"`
werden aus `app.go` in benannte Konstanten gezogen (bisher als Literale dreifach dupliziert) und
von `registerGlobalHotkeys()` **und** `dictation_mic.go` verwendet:

| Physische Taste | Bit | Frontend-Event |
|---|---|---|
| Record | `RECORD` | `hotkey:toggle-record` |
| Stop | `STOP` | `hotkey:toggle-record` |
| Rücklauf | `REWIND` | `hotkey:toggle-pause` |
| Insert/Overwrite | `INS_OVR` | `hotkey:accept-insert` |

Da es sich um exakt dieselben Wails-Events handelt, die `Recorder.tsx` bereits über `EventsOn`
konsumiert, sind **keine Frontend-Änderungen** nötig.

## Verifikations-Tooling (für später, sobald Hardware da ist)

Env-Var `ODX_DICTATION_MIC_DEBUG=1` aktiviert `log.Printf`-Ausgabe der rohen Report-Bytes (Hex)
für jedes erkannte Gerät, bevor sie interpretiert werden. Damit lässt sich der tatsächliche
Byte-Offset/das Report-ID-Verhalten direkt gegen die obige Tabelle prüfen, ohne Code ändern zu
müssen — das ist das im Todo geforderte Verifikationswerkzeug.

## Neue Abhängigkeit

`github.com/karalabe/hid` (BSD-3-Clause). Auf Linux zusätzlich zur Laufzeit/Buildzeit
`libusb-1.0-0-dev` nötig (analog zur bereits dokumentierten `webkit2gtk-4.1`/`gtk+-3.0`-Abhängigkeit
in `webview_perms.go`) — wird in `AGENTS.md`/`CLAUDE.md` ergänzt.

## Out of Scope (bewusst nicht Teil dieser Umsetzung)

- LED-Steuerung am SpeechMike (`SET_LED`-Command) — nicht gebraucht, nur Tasten lesen.
- SpeechMike Browser/Gamepad-Modus, Foot-Control-Pedale — im Todo nicht gefordert.
- Usage-Page/Usage-Filterung bei der Geräteerkennung (s.o., Plattformlimitierung von
  `karalabe/hid`).
- Automatisches Deaktivieren einzelner Buttons — alle vier sind ab Merge aktiv, auch ungetestet;
  das ist eine bewusste Nutzerentscheidung (keine Hardware verfügbar, Umsetzung erfolgt trotzdem).

## Testing

Kein Test-Setup im Repo (siehe AGENTS.md). Verifikation beschränkt sich auf:
- `go build ./...` (bzw. `go vet`) für Compile-Korrektheit.
- Manuelle Prüfung, dass die App ohne angeschlossenes Gerät und ohne `libusb` weiterhin normal
  startet und die Tastatur-Hotkeys funktionieren.
- Echte Funktionsverifikation der Bitmasken/Offsets ist explizit vertagt, bis physische Hardware
  verfügbar ist (siehe Verifikations-Tooling oben).
