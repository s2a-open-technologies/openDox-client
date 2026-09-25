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

import (
	"log"
	"os/exec"
)

// notifyReview zeigt eine native Desktop-Benachrichtigung als Erinnerung, den
// eingefügten Text vor Verwendung zu prüfen. macOS-Äquivalent zu notify-send
// (Linux) via osascript — kein zusätzliches Paket nötig, da osascript Teil
// von macOS ist. Best-effort: Fehler werden nur geloggt.
func notifyReview() {
	script := `display notification "Text eingefügt – bitte vor Verwendung prüfen." with title "openDox"`
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		log.Printf("[opendox] osascript notification failed: %v", err)
	}
}
