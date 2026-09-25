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
	"unsafe"
)

const (
	gwlExStyle      = -20
	wsExNoActivate  = 0x08000000
	swpNoMove       = 0x0002
	swpNoSize       = 0x0001
	swpNoZOrder     = 0x0004
	swpNoActivate   = 0x0010
	swpFrameChanged = 0x0020
)

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procFindWindowExW            = user32.NewProc("FindWindowExW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procGetWindowLongPtrW        = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLongPtrW        = user32.NewProc("SetWindowLongPtrW")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
)

// ownWindow sucht das Wails-Hauptfenster dieses Prozesses (Klasse "wailsWindow",
// siehe wails internal/frontend/desktop/windows/window.go). Wails gibt das HWND
// nicht heraus; der PID-Abgleich verhindert, dass eine zweite laufende Instanz
// getroffen wird.
func ownWindow() uintptr {
	class, _ := syscall.UTF16PtrFromString("wailsWindow")
	pid := uint32(os.Getpid())
	var hwnd uintptr
	for {
		hwnd, _, _ = procFindWindowExW.Call(0, hwnd, uintptr(unsafe.Pointer(class)), 0)
		if hwnd == 0 {
			return 0
		}
		var wpid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&wpid)))
		if wpid == pid {
			return hwnd
		}
	}
}

// setOverlayNoActivate setzt/entfernt WS_EX_NOACTIVATE am Overlay. Mit gesetztem
// Stil holt ein Mausklick auf openDox den Fokus nicht aus dem Zielprogramm — der
// Button kann dann denselben Weg nehmen wie der Hotkey (kein Hide/Show). Gibt
// zurück, ob der Stil jetzt tatsächlich im gewünschten Zustand ist.
func setOverlayNoActivate(enabled bool) bool {
	hwnd := ownWindow()
	if hwnd == 0 {
		log.Printf("[opendox] setOverlayNoActivate: eigenes Fenster nicht gefunden")
		return false
	}
	// GetWindowLongPtrW liefert einen LONG_PTR; negative Indizes als uintptr übergeben.
	idx := gwlExStyle
	style, _, _ := procGetWindowLongPtrW.Call(hwnd, uintptr(idx))
	if enabled {
		style |= wsExNoActivate
	} else {
		style &^= wsExNoActivate
	}
	procSetWindowLongPtrW.Call(hwnd, uintptr(idx), style)
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0,
		swpNoMove|swpNoSize|swpNoZOrder|swpNoActivate|swpFrameChanged)
	log.Printf("[opendox] setOverlayNoActivate(%v) ok", enabled)
	return true
}
