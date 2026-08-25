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

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices
#import <ApplicationServices/ApplicationServices.h>

// kVK_ANSI_V ist der physische (layoutunabhängige) virtuelle Tastencode der
// "V"-Taste auf einem ANSI-Keyboard — Apples eigene HIToolbox-Konstante, hier
// hartkodiert statt per Header eingebunden, um keine zusätzliche Abhängigkeit
// zu brauchen.
#define KVK_ANSI_V 0x09

// simulateCmdV sendet Cmd+V über CGEventPost mit fest codiertem Tastencode.
// BEWUSST NICHT über robotgo.KeyTap: das löst intern TIS/TSM-Tastaturlayout-
// Auflösung (keyCodeForChar, um von einem Zeichen wie 'v' auf den aktuellen
// physischen Code zu kommen) aus, und laut Apple (siehe go-vgo/robotgo#341)
// sind TIS/TSM-Aufrufe AUSSCHLIESSLICH vom Main-Thread aus erlaubt — undefined
// behaviour sonst, bei uns reproduzierbar ein harter SIGTRAP-Crash (siehe
// go-vgo/robotgo#690), weil InsertTranscript in einem Wails-Dispatcher-
// Goroutine läuft, nicht auf dem Main-Thread. CGEventPost/CGEventCreate-
// KeyboardEvent sind reine Quartz-Event-Services-Aufrufe (IPC zum WindowServer)
// ohne diese Main-Thread-Einschränkung und brauchen keine Zeichen-Auflösung,
// weil der Tastencode für Cmd+V ohnehin immer derselbe ist (die physische
// "V"-Position, unabhängig vom Tastaturlayout — genau das erwarten Nutzer von
// einem Copy/Paste-Shortcut).
void simulateCmdV() {
    CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);

    CGEventRef keyVDown = CGEventCreateKeyboardEvent(source, (CGKeyCode)KVK_ANSI_V, true);
    CGEventSetFlags(keyVDown, kCGEventFlagMaskCommand);
    CGEventRef keyVUp = CGEventCreateKeyboardEvent(source, (CGKeyCode)KVK_ANSI_V, false);
    CGEventSetFlags(keyVUp, kCGEventFlagMaskCommand);

    CGEventPost(kCGHIDEventTap, keyVDown);
    CGEventPost(kCGHIDEventTap, keyVUp);

    CFRelease(keyVDown);
    CFRelease(keyVUp);
    CFRelease(source);
}
*/
import "C"

// simulatePaste löst Cmd+V aus — Ersatz für robotgo.KeyTap("v", "command") auf
// macOS, siehe Kommentar in simulateCmdV() oben für den Grund.
func simulatePaste() {
	C.simulateCmdV()
}
