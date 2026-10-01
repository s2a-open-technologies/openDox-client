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

//go:build linux

package main

/*
#cgo pkg-config: x11
#include <X11/Xlib.h>
#include <X11/keysym.h>
#include <string.h>

// modifiersHeld fragt per XQueryKeymap den kompletten Tastatur-Zustand ab (kein
// Event, keine Interaktion mit dem Fokus) und prüft, ob einer der Modifier noch
// physisch gedrückt ist — z.B. weil der Nutzer den globalen Hotkey Ctrl+Alt+V noch
// nicht ganz losgelassen hat. Öffnet für die eine Abfrage eine eigene, kurzlebige
// Display-Verbindung statt eine dauerhafte zu halten, weil InsertTranscript
// (Wails-Dispatcher-Goroutine) nicht der Thread ist, der die Haupt-X11-Verbindung
// des Prozesses hält.
int modifiersHeld() {
    Display *d = XOpenDisplay(NULL);
    if (d == NULL) {
        return 0;
    }

    char keys[32];
    XQueryKeymap(d, keys);

    KeySym syms[] = {
        XK_Control_L, XK_Control_R,
        XK_Alt_L, XK_Alt_R,
        XK_Shift_L, XK_Shift_R,
        XK_Super_L, XK_Super_R,
    };

    int held = 0;
    for (size_t i = 0; i < sizeof(syms) / sizeof(syms[0]); i++) {
        KeyCode kc = XKeysymToKeycode(d, syms[i]);
        if (kc != 0 && (keys[kc / 8] & (1 << (kc % 8)))) {
            held = 1;
            break;
        }
    }

    XCloseDisplay(d);
    return held;
}
*/
import "C"

import (
	"log"
	"time"

	"github.com/go-vgo/robotgo"
)

// insertText wartet, bis Strg/Alt/Shift/Super physisch losgelassen sind, und
// simuliert dann Strg+V über robotgo (XTest, dieselbe Bibliothek, die schon
// robotgo.Type nutzt). Ein simuliertes Ctrl+V wurde hier früher direkt nach dem
// Hotkey-Trigger ausprobiert und kollidierte mit dessen noch nicht losgelassenem
// Ctrl+Alt — das explizite Warten behebt genau das.
func insertText(text string) error {
	waitModifiersReleased(2 * time.Second)
	return robotgo.KeyTap("v", "control")
}

func waitModifiersReleased(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for C.modifiersHeld() != 0 {
		if time.Now().After(deadline) {
			log.Printf("[opendox] insertText: Modifier nach %v noch gedrückt", timeout)
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
	return true
}
