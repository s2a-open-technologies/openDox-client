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

//go:build windows

package main

import (
	"log"

	toast "git.sr.ht/~jackmordaunt/go-toast/v2"
)

// notifyReview zeigt eine native Desktop-Benachrichtigung als Erinnerung, den
// eingefügten Text vor Verwendung zu prüfen. Windows-Äquivalent zu notify-send
// (Linux). git.sr.ht/~jackmordaunt/go-toast war bereits als transitive
// Abhängigkeit über Wails selbst vorhanden (dessen eigene Windows-Notification-API),
// daher kein neues Modul nötig. Fällt ohne registrierte AppUserModelID
// automatisch auf einen PowerShell-Toast zurück, statt zu scheitern. Best-effort:
// Fehler werden nur geloggt.
func notifyReview() {
	n := toast.Notification{
		AppID: "openDox",
		Title: "openDox",
		Body:  "Text eingefügt – bitte vor Verwendung prüfen.",
	}
	if err := n.Push(); err != nil {
		log.Printf("[opendox] toast notification failed: %v", err)
	}
}
