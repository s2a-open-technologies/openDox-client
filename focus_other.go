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

//go:build !windows

package main

// startForegroundWatcher/restoreLastForegroundWindow sind auf macOS/Linux No-Op-Stubs
// — siehe focus_windows.go für den Grund, warum Windows das explizit braucht
// (WindowHide() dort verlässt sich nicht zuverlässig genug auf die automatische
// Fokus-Rückgabe des Betriebssystems). Auf Linux/macOS hat sich das bisherige
// Hide-and-hope-Verhalten (siehe Kommentare in InsertTranscript) als ausreichend
// erwiesen.
func startForegroundWatcher()      {}
func restoreLastForegroundWindow() {}
