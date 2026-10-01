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

//go:build windows

// Package winfocus ermittelt, welches Programm gerade im Vordergrund ist, also wohin
// ein Einfügen per Strg+V ginge. Wie winclip ohne Wails/cgo-Abhängigkeit.
package winfocus

import (
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// maxTitleUnits begrenzt die gelesene Titellänge (UTF-16-Einheiten).
const maxTitleUnits = 200

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	procGetWindowTextW    = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW = user32.NewProc("GetWindowTextLengthW")
)

// Foreground liefert den kleingeschriebenen .exe-Namen und den Fenstertitel des
// Vordergrundfensters. Leere Strings, wenn es keins gibt, sich der Prozess nicht
// ermitteln lässt oder das Vordergrundfenster der Client selbst ist (dann ist das
// Einfügeziel unbekannt). Der Titel enthält bei Klinik-Software oft Patientennamen:
// nur anzeigen, nie loggen oder speichern.
func Foreground() (process, title string) {
	hwnd := windows.GetForegroundWindow()
	if hwnd == 0 {
		return "", ""
	}
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(hwnd, &pid); err != nil || pid == 0 {
		return "", ""
	}
	if pid == uint32(os.Getpid()) {
		return "", ""
	}
	return processName(pid), windowTitle(hwnd)
}

func processName(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)

	buf := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return ""
	}
	return strings.ToLower(filepath.Base(windows.UTF16ToString(buf[:size])))
}

func windowTitle(hwnd windows.HWND) string {
	n, _, _ := procGetWindowTextLenW.Call(uintptr(hwnd))
	if n == 0 {
		return ""
	}
	if n > maxTitleUnits {
		n = maxTitleUnits
	}
	buf := make([]uint16, n+1)
	r, _, _ := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r == 0 {
		return ""
	}
	return windows.UTF16ToString(buf[:r])
}
