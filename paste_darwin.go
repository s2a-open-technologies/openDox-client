// openDox - Medical transcription platform
// Copyright (C) 2026  S2A Open Technologies gGmbH
// SPDX-FileCopyrightText: 2026 S2A Open Technologies gGmbH
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

// Physische (layoutunabhängige) virtuelle Tastencodes der Modifier, beide Seiten
// je Taste — Apples eigene HIToolbox-Konstanten, hartkodiert aus demselben Grund
// wie KVK_ANSI_V oben.
static const CGKeyCode modifierKeycodes[] = {
    0x37, 0x36, // Command links/rechts
    0x3A, 0x3D, // Option links/rechts
    0x3B, 0x3E, // Control links/rechts
    0x38, 0x3C, // Shift links/rechts
};

// modifiersHeld prüft per CGEventSourceKeyState (reiner Zustandsabruf, kein Event,
// keine TIS/TSM-Auflösung) ob noch ein Modifier physisch gedrückt ist — z.B. weil
// der Nutzer Cmd+Opt+V (unseren Hotkey) noch nicht ganz losgelassen hat. Ohne diese
// Prüfung würde ein sofortiges Cmd+V-Paste danach zu Cmd+Opt+V werden.
int modifiersHeld() {
    for (size_t i = 0; i < sizeof(modifierKeycodes) / sizeof(modifierKeycodes[0]); i++) {
        if (CGEventSourceKeyState(kCGEventSourceStateHIDSystemState, modifierKeycodes[i])) {
            return 1;
        }
    }
    return 0;
}
*/
import "C"

// simulatePaste löst Cmd+V aus — Ersatz für robotgo.KeyTap("v", "command") auf
// macOS, siehe Kommentar in simulateCmdV() oben für den Grund.
func simulatePaste() {
	C.simulateCmdV()
}

// modifiersHeld meldet, ob aktuell noch ein Modifier (Cmd/Option/Control/Shift)
// physisch gedrückt ist, siehe modifiersHeld() oben.
func modifiersHeld() bool {
	return C.modifiersHeld() != 0
}
