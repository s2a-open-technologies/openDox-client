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

//go:build darwin

package main

import (
	"os"
	"syscall"
)

// silenceNativeConsole leitet fd 1 und fd 2 auf /dev/null um, damit native
// Bibliotheken still sind. Muss VOR dem Laden der betroffenen Bibliotheken
// laufen — daher der Aufruf als allererste Aktion in main().
func silenceNativeConsole() {
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return
	}
	_ = syscall.Dup2(int(devNull.Fd()), 1)
	_ = syscall.Dup2(int(devNull.Fd()), 2)
}

// quietEnv entfernt Debug-Ausgaben aktivierende Umgebungsvariablen.
func quietEnv() {
	os.Unsetenv("G_MESSAGES_DEBUG")
	os.Unsetenv("GTK_DEBUG")
	os.Unsetenv("WEBKIT_DEBUG")
}

// preSilence wird vor allen nativen Initialisierungen aufgerufen.
func preSilence() {
	quietEnv()
}

// ensureDebugConsole ist auf macOS ein No-Op-Stub — anders als das Windows-GUI-
// Subsystem-Binary (siehe console_silence_windows.go) läuft dieser Build immer mit
// einer vom startenden Terminal geerbten Konsole, os.Stdout/os.Stderr sind daher
// schon nutzbar.
func ensureDebugConsole() {}
