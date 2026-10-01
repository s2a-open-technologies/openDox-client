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

import (
	"log"
	"os/exec"
)

// notifyReview zeigt eine native Desktop-Benachrichtigung als Erinnerung, den
// eingefügten Text vor Verwendung zu prüfen. Best-effort: falls notify-send
// fehlt oder fehlschlägt, wird das nur geloggt, da die Zwischenablage/der
// Zieltext bereits erfolgreich gesetzt wurde.
func notifyReview() {
	cmd := exec.Command("notify-send", "openDox", "Text eingefügt – bitte vor Verwendung prüfen.")
	if err := cmd.Run(); err != nil {
		log.Printf("[opendox] notify-send failed: %v", err)
	}
}
