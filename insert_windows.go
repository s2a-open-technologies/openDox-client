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

package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-vgo/robotgo"
)

// Pause zwischen den SendInput-Unicode-Events beim Tippen (nur Citrix-Pfad).
// Ohne Pause verschluckt/verstümmelt z.B. das Windows-11-Notepad den Burst.
const typeCharDelayMs = 10

// Citrix-Clients: Die Zwischenablage kommt in der Sitzung oft nicht an (Umleitung
// ist in Klinik-Umgebungen häufig gesperrt), getippte Tasten dagegen schon.
var typeOnlyProcesses = map[string]bool{
	"wfica32.exe":  true,
	"cdviewer.exe": true,
}

const (
	vkShift   = 0x10
	vkControl = 0x11
	vkMenu    = 0x12
	vkLWin    = 0x5B
	vkRWin    = 0x5C
	vkV       = 0x56

	inputKeyboard  = 1
	keyeventfKeyUp = 0x0002

	processQueryLimitedInformation = 0x1000
)

var (
	procGetAsyncKeyState          = user32.NewProc("GetAsyncKeyState")
	procGetForegroundWindow       = user32.NewProc("GetForegroundWindow")
	procSendInput                 = user32.NewProc("SendInput")
	kernel32Insert                = syscall.NewLazyDLL("kernel32.dll")
	procQueryFullProcessImageName = kernel32Insert.NewProc("QueryFullProcessImageNameW")
)

// INPUT-Struktur für 64-bit: type + Padding, KEYBDINPUT, Auffüllen auf die
// Größe der Union (MOUSEINPUT, 32 Byte) — insgesamt 40 Byte wie in der WinAPI.
type keybdInput struct {
	wVk         uint16
	wScan       uint16
	dwFlags     uint32
	time        uint32
	dwExtraInfo uintptr
}

type keyboardInputEvent struct {
	typ uint32
	ki  keybdInput
	_   [8]byte
}

// insertText bringt den Text ins Vordergrundfenster: Einfügen per Strg+V (exakt,
// in einem Rutsch — beim Tippen können Zeichen still verloren gehen oder doppelt
// ankommen), außer bei Citrix, wo nur getippte Tasten zuverlässig ankommen. Die
// Zwischenablage hat InsertTranscript bereits gesetzt.
func insertText(text string) error {
	// Noch gehaltene Hotkey-Modifier würden aus Strg+V ein Strg+Alt+V (unseren
	// eigenen Hotkey) bzw. beim Tippen AltGr-Zeichen/Shortcuts machen.
	if !waitModifiersReleased(2 * time.Second) {
		return fmt.Errorf("Strg/Alt/Shift noch gedrückt – Text liegt in der Zwischenablage")
	}

	exe := foregroundProcessName()
	if typeOnlyProcesses[exe] {
		log.Printf("[opendox] insertText: tippe (Vordergrund: %s)", exe)
		robotgo.Type(text, 0, typeCharDelayMs)
		return nil
	}
	log.Printf("[opendox] insertText: füge per Strg+V ein (Vordergrund: %s)", exe)
	return sendCtrlV()
}

func waitModifiersReleased(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		held := false
		for _, vk := range []uintptr{vkShift, vkControl, vkMenu, vkLWin, vkRWin} {
			state, _, _ := procGetAsyncKeyState.Call(vk)
			if state&0x8000 != 0 {
				held = true
				break
			}
		}
		if !held {
			return true
		}
		if time.Now().After(deadline) {
			log.Printf("[opendox] insertText: Modifier nach %v noch gedrückt", timeout)
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// foregroundProcessName liefert den kleingeschriebenen .exe-Namen des Prozesses
// hinter dem Vordergrundfenster, "" wenn er sich nicht ermitteln lässt.
func foregroundProcessName() string {
	hwnd, _, _ := procGetForegroundWindow.Call()
	if hwnd == 0 {
		return ""
	}
	var pid uint32
	procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if pid == 0 {
		return ""
	}
	h, err := syscall.OpenProcess(processQueryLimitedInformation, false, pid)
	if err != nil {
		return ""
	}
	defer syscall.CloseHandle(h)

	buf := make([]uint16, syscall.MAX_PATH)
	size := uint32(len(buf))
	ret, _, _ := procQueryFullProcessImageName.Call(uintptr(h), 0,
		uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret == 0 {
		return ""
	}
	return strings.ToLower(filepath.Base(syscall.UTF16ToString(buf[:size])))
}

func sendCtrlV() error {
	inputs := []keyboardInputEvent{
		{typ: inputKeyboard, ki: keybdInput{wVk: vkControl}},
		{typ: inputKeyboard, ki: keybdInput{wVk: vkV}},
		{typ: inputKeyboard, ki: keybdInput{wVk: vkV, dwFlags: keyeventfKeyUp}},
		{typ: inputKeyboard, ki: keybdInput{wVk: vkControl, dwFlags: keyeventfKeyUp}},
	}
	n, _, err := procSendInput.Call(uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])), unsafe.Sizeof(inputs[0]))
	if int(n) != len(inputs) {
		return fmt.Errorf("SendInput(Strg+V) fehlgeschlagen: %v", err)
	}
	return nil
}
