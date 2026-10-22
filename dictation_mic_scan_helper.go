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
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/ricknicius99/hid"
)

// dictationMicScanResult ist die über den Scan-Subprozess (siehe unten) ausgetauschte,
// abgespeckte Sicht auf ein gefundenes Diktiermikro-Gerät.
type dictationMicScanResult struct {
	Path      string `json:"path"`
	VendorID  uint16 `json:"vendor_id"`
	ProductID uint16 `json:"product_id"`
}

// runDictationMicScanHelper führt je Diktiermikro-Hersteller einen hid.Enumerate()-Aufruf aus und schreibt passende
// Diktiermikro-Geräte zeilenweise als JSON auf stdout. Wird von main() aufgerufen, wenn die
// Umgebungsvariable ODX_DICTATION_MIC_SCAN_HELPER gesetzt ist (siehe scanDictationMicDevices
// unten, das diesen Modus als Subprozess startet) statt der normalen GUI.
//
// Grund für den Subprozess: hid_enumerate()s macOS-Backend (third_party/hid/hidapi_mac.h) kann
// auf macOS 26 nicht-deterministisch mit einem fatalen, nicht abfangbaren EXC_GUARD/
// GUARD_TYPE_MACH_PORT crashen (vermutlich eine Race tief in Apples eigenem IOKit zwischen
// IOHIDManagerCopyDevices()' Momentaufnahme und dem anschließenden IORegistryEntryGetPath() pro
// Gerät — siehe Commit-Historie für zwei vorherige, für sich genommen korrekte, aber allein
// nicht ausreichende Fixes). Läuft der riskante Aufruf in einem eigenen, jederzeit
// wegwerfbaren Subprozess statt im Hauptprozess, kostet ein Absturz nur einen verpassten
// Scan-Zyklus statt der ganzen App samt einer eventuell laufenden Diktier-Sitzung.
//
// Aber auch der Absturz des Subprozesses ist sichtbar: macOS zeigt dafür den Dialog „openDox
// wurde unerwartet beendet“, obwohl die App weiterläuft. Daher wird nur nach den Hersteller-IDs
// der Diktiermikros gesucht: hid_enumerate() filtert nach VID, BEVOR es IORegistryEntryGetPath()
// aufruft, die Pfade fremder Geräte (Tastatur, Trackpad, Bluetooth, virtuelle HIDs), bei denen
// der Absturz auftritt, werden also gar nicht mehr gelesen.
func runDictationMicScanHelper() {
	var infos []hid.DeviceInfo
	for _, vid := range dictationMicVendorIDs {
		infos = append(infos, hid.Enumerate(vid, 0)...)
	}
	enc := json.NewEncoder(os.Stdout)
	for _, info := range infos {
		if matchDictationMicProfile(info.VendorID, info.ProductID) == nil {
			continue
		}
		_ = enc.Encode(dictationMicScanResult{
			Path:      info.Path,
			VendorID:  info.VendorID,
			ProductID: info.ProductID,
		})
	}
}

// scanDictationMicDevices startet sich selbst als Subprozess im Scan-Helper-Modus (siehe oben),
// wartet auf dessen Ergebnis oder timeout, und liefert die gefundenen Geräte zurück. Stürzt der
// Subprozess ab (z.B. wegen des oben beschriebenen EXC_GUARD) oder überschreitet er timeout,
// wird das nur geloggt — der nächste periodische Scan versucht es einfach erneut.
func scanDictationMicDevices(timeout time.Duration) []dictationMicScanResult {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("[opendox] dictation mic: os.Executable() fehlgeschlagen: %v", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, exe)
	cmd.Env = append(os.Environ(), "ODX_DICTATION_MIC_SCAN_HELPER=1")
	out, err := cmd.Output()
	if err != nil {
		log.Printf("[opendox] dictation mic: Scan-Subprozess fehlgeschlagen (%v), nächster Versuch in %s", err, dictationMicRescanInterval)
		return nil
	}

	var results []dictationMicScanResult
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var r dictationMicScanResult
		if decErr := dec.Decode(&r); decErr != nil {
			break
		}
		results = append(results, r)
	}
	return results
}
