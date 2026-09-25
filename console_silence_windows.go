// openDox - Medical transcription platform
// Copyright (C) 2026  Christopher Abanilla <christopher.abanilla@gmx.de>
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

//go:build windows

package main

import (
	"log"
	"os"
	"syscall"
)

// Windows-GUI-Builds schreiben keine nativen Konsolen-Meldungen (kein
// GTK/WebKitGTK/JSC — WebView2 läuft out-of-process), dort stillstellt
// bereits setupConsole() über log.SetOutput. Alle Hooks hier No-ops.

func silenceNativeConsole() {}
func quietEnv()             {}
func preSilence()           {}

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procAllocConsole = kernel32.NewProc("AllocConsole")
)

// ensureDebugConsole allokiert bei --debug ein sichtbares Konsolenfenster, falls der
// Prozess noch keines hat. Der Windows-Build ist ein GUI-Subsystem-Binary (kein
// Konsolen-Fenster beim Start per Doppelklick/Verknüpfung) — os.Stdout/os.Stderr zeigen
// dann auf ungültige Handles, log.Printf landet also nirgendwo sichtbar, selbst mit
// --debug. AllocConsole() schlägt mit ERROR_ACCESS_DENIED fehl, wenn bereits eine
// Konsole angehängt ist (z.B. aus cmd.exe/PowerShell gestartet) — dort bleibt alles
// unverändert, weil os.Stdout/os.Stderr dort schon auf die vererbte Konsole zeigen.
func ensureDebugConsole() {
	ret, _, _ := procAllocConsole.Call()
	if ret == 0 {
		return
	}
	if f, err := os.OpenFile("CONOUT$", os.O_RDWR, 0); err == nil {
		os.Stdout = f
		os.Stderr = f
		log.SetOutput(f)
	}
	if f, err := os.OpenFile("CONIN$", os.O_RDWR, 0); err == nil {
		os.Stdin = f
	}
}
