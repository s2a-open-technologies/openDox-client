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

import (
	"log"
	"time"
)

// insertText wartet, bis Cmd/Option/Control/Shift physisch losgelassen sind (siehe
// paste_darwin.go), und löst dann simulatePaste() (Cmd+V) aus. Noch gehaltene
// Modifier würden aus Cmd+V sonst z.B. unseren eigenen Cmd+Opt+V-Hotkey machen.
func insertText(text string) error {
	waitModifiersReleased(2 * time.Second)
	simulatePaste()
	return nil
}

func waitModifiersReleased(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for modifiersHeld() {
		if time.Now().After(deadline) {
			log.Printf("[opendox] insertText: Modifier nach %v noch gedrückt", timeout)
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
	return true
}
