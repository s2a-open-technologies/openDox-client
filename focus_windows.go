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
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procFindWindowW         = user32.NewProc("FindWindowW")
)

// ownHWND wird beim ersten erfolgreichen Poll in startForegroundWatcher per
// FindWindowW anhand des Fenstertitels ermittelt (siehe Title: "openDox" in
// main.go) und danach nicht mehr neu gesucht.
var ownHWND uintptr

// lastForeignHWND ist das zuletzt beobachtete Vordergrundfenster, das NICHT
// openDox selbst war — also der wahrscheinliche Insert-Zielfenster-Kandidat.
var lastForeignHWND atomic.Uintptr

// startForegroundWatcher pollt alle 200ms das aktuelle Vordergrundfenster und merkt
// sich in lastForeignHWND das letzte, das nicht openDox selbst war.
//
// Grund: WindowHide() (winc.ControlBase.Hide(), reines ShowWindow(hwnd, SW_HIDE))
// verlässt sich implizit darauf, dass Windows nach dem Verstecken automatisch das
// vorherige Fenster aktiviert - das ist auf Windows 11 in der Praxis nicht
// zuverlässig genug (bestätigt: Button-Insert liefert dort konsequent NICHTS,
// während derselbe Tastatur-Injektionscode über den Hotkey-Pfad - der nie versteckt
// - zuverlässig funktioniert). InsertTranscript ruft stattdessen nach dem Hide
// explizit restoreLastForegroundWindow() auf, um das zuvor aktive Fenster gezielt
// zurückzuholen, statt auf den impliziten OS-Mechanismus zu hoffen.
//
// SetForegroundWindow() darf laut Microsoft-Doku vom aktuellen Eigentümer des
// Vordergrundfensters aufgerufen werden, ohne durch den üblichen Fokus-Diebstahl-
// Schutz blockiert zu werden - genau dieser Fall liegt hier vor, da openDox beim
// Klick auf "Akzeptieren und einfügen" selbst gerade das Vordergrundfenster ist.
func startForegroundWatcher() {
	go func() {
		for {
			time.Sleep(200 * time.Millisecond)

			if ownHWND == 0 {
				titlePtr, err := syscall.UTF16PtrFromString("openDox")
				if err != nil {
					continue
				}
				h, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
				if h == 0 {
					continue
				}
				ownHWND = h
			}

			fg, _, _ := procGetForegroundWindow.Call()
			if fg != 0 && fg != ownHWND {
				lastForeignHWND.Store(fg)
			}
		}
	}()
}

// restoreLastForegroundWindow holt das zuletzt beobachtete Nicht-openDox-Fenster
// (siehe startForegroundWatcher) aktiv in den Vordergrund. No-Op, falls seit dem
// App-Start noch nie ein anderes Fenster im Vordergrund war.
func restoreLastForegroundWindow() {
	hwnd := lastForeignHWND.Load()
	if hwnd == 0 {
		return
	}
	ret, _, err := procSetForegroundWindow.Call(hwnd)
	if ret == 0 {
		log.Printf("[opendox] SetForegroundWindow(%x) fehlgeschlagen: %v", hwnd, err)
	}
}
