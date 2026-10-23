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

//go:build darwin

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"time"
)

// scanDictationMicDevices startet sich selbst als Subprozess im Scan-Helper-Modus (siehe
// runDictationMicScanHelper), wartet auf dessen Ergebnis oder timeout, und liefert die gefundenen
// Geräte zurück. Stürzt der Subprozess ab (z.B. wegen des dort beschriebenen EXC_GUARD) oder
// überschreitet er timeout, wird das nur geloggt — der nächste periodische Scan versucht es einfach
// erneut. worker bleibt hier ungenutzt: Enumerate läuft im Subprozess, nicht im Hauptprozess.
func scanDictationMicDevices(_ *dictationMicWorker, timeout time.Duration) []dictationMicScanResult {
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
