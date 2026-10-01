// openDox - Medical transcription platform
// Copyright (C) 2026  Christopher Abanilla <christopher.abanilla@gmx.de>
// SPDX-FileCopyrightText: 2026 Christopher Abanilla <christopher.abanilla@gmx.de>
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
	"io"
	"log"
	"os"
)

// setupConsole schaltet die Konsolen-Ausgabe diesermain-Phase um: Mit --debug
// (oder in Dev-Builds, damit `wails dev` nicht blind läuft) bleibt alles wie
// bisher. Ohne den Flag wird die komplette Ausgabe stillgestellt — sowohl das
// Go-log-Paket als auch native Meldungen von GTK/WebKitGTK/JSC (z.B.
// "Overriding existing handler for signal 10"), die direkt auf fd 1/2 schreiben
// und an log.SetOutput vorbeigehen.
//
// Muss NACH dem Scan-Helper-Check aufgerufen werden: Der Helper-Modus
// (ODX_DICTATION_MIC_SCAN_HELPER) liefert sein JSON über stdout an den
// Elternprozess und darf niemals umgeleitet werden.
func setupConsole() {
	if debugRequested() {
		return
	}
	log.SetOutput(io.Discard)
	silenceNativeConsole()
}

// debugRequested liefert true, wenn --debug/-debug übergeben wurde oder der
// Build ein Dev-Build ist (wails dev / go run -tags dev).
func debugRequested() bool {
	if isDevBuild {
		return true
	}
	for _, arg := range os.Args[1:] {
		if arg == "--debug" || arg == "-debug" {
			return true
		}
	}
	return false
}
