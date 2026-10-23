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

// dictationMicVendorIDs sind alle USB-Vendor-IDs aus matchDictationMicProfile (Philips, Nuance).
// Der Scan fragt nur diese ab, damit unter Windows keine fremden HID-Geräte (Maus, Tastatur, ...)
// geöffnet werden — siehe enumerateDictationMics.
var dictationMicVendorIDs = []uint16{0x0911, 0x0554}

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

// fallingEdge liefert die Bits, die in cur gegenüber prev gelöscht wurden (1->0-Übergang) — das
// Loslassen einer Taste.
func fallingEdge(prev, cur micButtonMask) micButtonMask {
	return prev &^ cur
}
