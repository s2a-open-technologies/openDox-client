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

// Package winclip setzt und leert die Windows-Zwischenablage mit den Markierungen,
// die Verlauf (Win+V) und Cloud-Clipboard ausschließen. Wails' ClipboardSetText
// setzt nur Klartext und kann das nicht. Bewusst ohne Wails/cgo-Abhängigkeit, damit
// das Paket auch auf Nicht-Windows-Rechnern per GOOS=windows kompiliert und geprüft
// werden kann.
package winclip

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002

	openTimeout  = time.Second
	openInterval = 5 * time.Millisecond
)

// Registrierte Formate, deren Namen Windows kennt (Microsoft Learn: "Clipboard
// Formats" → "Cloud Clipboard and Clipboard History Formats"):
//   - ExcludeClipboardContentFromMonitorProcessing: beliebige Daten; schließt Verlauf
//     UND Synchronisation aus.
//   - CanIncludeInClipboardHistory: DWORD 0 schließt den Verlauf aus.
//   - CanUploadToCloudClipboard: DWORD 0 schließt die Synchronisation aus.
const (
	formatExcludeFromMonitor  = "ExcludeClipboardContentFromMonitorProcessing"
	formatCanIncludeInHistory = "CanIncludeInClipboardHistory"
	formatCanUploadToCloud    = "CanUploadToCloudClipboard"
)

var (
	user32   = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")

	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procEmptyClipboard             = user32.NewProc("EmptyClipboard")
	procSetClipboardData           = user32.NewProc("SetClipboardData")
	procRegisterClipboardFormatW   = user32.NewProc("RegisterClipboardFormatW")
	procGetClipboardSequenceNumber = user32.NewProc("GetClipboardSequenceNumber")

	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
	procGlobalFree   = kernel32.NewProc("GlobalFree")
)

// SetText ersetzt den Inhalt der Zwischenablage durch text (CF_UNICODETEXT, keine
// Zeilenumbruch-Umwandlung — wie zuvor Wails). Mit sensitive wird der Eintrag
// zusätzlich vom Zwischenablage-Verlauf und der Cloud-Synchronisation ausgenommen;
// gelingt das Markieren nicht, wird die Zwischenablage wieder geleert und ein Fehler
// zurückgegeben (fail closed: lieber nicht einfügen als unmarkiert liegen lassen).
// Rückgabe ist die Sequenznummer der Zwischenablage nach dem Setzen, für
// ClearIfUnchanged.
func SetText(text string, sensitive bool) (uint32, error) {
	// Open/Close müssen auf demselben OS-Thread laufen, sonst droht ein Deadlock
	// der Zwischenablage (siehe auch Wails' win32/clipboard.go).
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	units, err := windows.UTF16FromString(strings.ReplaceAll(text, "\x00", ""))
	if err != nil {
		return 0, err
	}
	textBytes := make([]byte, len(units)*2)
	for i, u := range units {
		binary.LittleEndian.PutUint16(textBytes[i*2:], u)
	}

	if err := waitOpen(); err != nil {
		return 0, err
	}
	closed := false
	defer func() {
		if !closed {
			procCloseClipboard.Call()
		}
	}()

	if r, _, callErr := procEmptyClipboard.Call(); r == 0 {
		return 0, fmt.Errorf("EmptyClipboard: %w", callErr)
	}
	if err := setData(cfUnicodeText, textBytes); err != nil {
		return 0, err
	}

	if sensitive {
		dwordZero := make([]byte, 4) // serialisiertes DWORD 0
		for _, name := range []string{formatExcludeFromMonitor, formatCanIncludeInHistory, formatCanUploadToCloud} {
			id, err := registerFormat(name)
			if err == nil {
				err = setData(uintptr(id), dwordZero)
			}
			if err != nil {
				procEmptyClipboard.Call() // unmarkierten Text nicht liegen lassen
				return 0, fmt.Errorf("Zwischenablage konnte nicht als vertraulich markiert werden: %w", err)
			}
		}
	}

	procCloseClipboard.Call()
	closed = true
	return SequenceNumber(), nil
}

// SequenceNumber liefert die aktuelle Sequenznummer der Zwischenablage; sie ändert
// sich bei jeder Änderung des Inhalts durch irgendein Programm.
func SequenceNumber() uint32 {
	r, _, _ := procGetClipboardSequenceNumber.Call()
	return uint32(r)
}

// ClearIfUnchanged leert die Zwischenablage nur, wenn ihre Sequenznummer noch seq
// ist, der Inhalt also weiterhin vom Client stammt — Kopiertes, das der Nutzer
// zwischenzeitlich selbst angelegt hat, bleibt erhalten. Meldet, ob geleert wurde.
func ClearIfUnchanged(seq uint32) bool {
	if seq == 0 || SequenceNumber() != seq {
		return false
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := waitOpen(); err != nil {
		return false
	}
	defer procCloseClipboard.Call()
	// Erneut prüfen, jetzt bei geöffneter (und damit für andere gesperrter) Zwischenablage.
	if SequenceNumber() != seq {
		return false
	}
	r, _, _ := procEmptyClipboard.Call()
	return r != 0
}

func waitOpen() error {
	deadline := time.Now().Add(openTimeout)
	for {
		r, _, err := procOpenClipboard.Call(0)
		if r != 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("Zwischenablage nicht verfügbar: %w", err)
		}
		time.Sleep(openInterval)
	}
}

func registerFormat(name string) (uint32, error) {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return 0, err
	}
	r, _, callErr := procRegisterClipboardFormatW.Call(uintptr(unsafe.Pointer(p)))
	if r == 0 {
		return 0, fmt.Errorf("RegisterClipboardFormat(%s): %w", name, callErr)
	}
	return uint32(r), nil
}

// setData legt data als GMEM_MOVEABLE-Speicherblock unter dem Format format auf die
// (geöffnete) Zwischenablage. Nach Erfolg gehört der Block dem System.
func setData(format uintptr, data []byte) error {
	h, _, err := procGlobalAlloc.Call(gmemMoveable, uintptr(len(data)))
	if h == 0 {
		return fmt.Errorf("GlobalAlloc: %w", err)
	}
	l, _, err := procGlobalLock.Call(h)
	if l == 0 {
		procGlobalFree.Call(h)
		return fmt.Errorf("GlobalLock: %w", err)
	}
	// l ist ein von GlobalLock gelieferter, gültiger Zeiger; der Umweg über
	// *(*unsafe.Pointer)(&l) vermeidet die vet-Warnung zu uintptr→unsafe.Pointer.
	dst := unsafe.Slice((*byte)(*(*unsafe.Pointer)(unsafe.Pointer(&l))), len(data))
	copy(dst, data)
	procGlobalUnlock.Call(h)

	if r, _, err := procSetClipboardData.Call(format, h); r == 0 {
		procGlobalFree.Call(h)
		return fmt.Errorf("SetClipboardData: %w", err)
	}
	return nil
}
