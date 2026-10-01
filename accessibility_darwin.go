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

//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework Cocoa
#import <ApplicationServices/ApplicationServices.h>
#import <Cocoa/Cocoa.h>

// checkAccessibility prüft still, ob der Prozess für Accessibility
// (Bedienhilfen) vertraut ist. Löst bewusst KEINEN Systemdialog aus —
// der Nutzer hat die Berechtigung schon erteilt, und bei jedem Neustart
// erneut aufdringlich zu sein (passiert bei wechselndem Binary-Hash,
// z.B. nach `wails build`) wäre nervig. Gibt 1 zurück, wenn vertraut.
int checkAccessibility() {
    return AXIsProcessTrusted() ? 1 : 0;
}
*/
import "C"

// ensureAccessibility prüft still, ob der Prozess für Accessibility
// (Bedienhilfen) vertraut ist. Liefert true, wenn vertraut — die App kann
// dann Tasten injizieren (robotgo.TypeStr/Paste). Bei false zeigt das
// Frontend einen Toast mit Hinweis auf die Systemeinstellungen; der
// aufdringliche Systemdialog wird bewusst nicht mehr ausgelöst, da
// macOS ihn bei jedem Binary-Wechsel neu zeigt und der Nutzer die
// Berechtigung in der Regel schon erteilt hat.
//
// Voraussetzung, damit die manuell in den Systemeinstellungen erteilte
// Freigabe hier überhaupt als true ankommt: die App muss mit einem echten,
// von Apple ausgestellten Zertifikat signiert sein (siehe build-macos.sh) —
// mit reiner ad-hoc- oder selbstsignierter Signatur hat AXIsProcessTrusted()
// auf aktuellem macOS (getestet: 26.4/arm64) empirisch dauerhaft false
// geliefert, obwohl der Schalter in den Systemeinstellungen sichtbar an war.
func ensureAccessibility() bool {
	return C.checkAccessibility() != 0
}
