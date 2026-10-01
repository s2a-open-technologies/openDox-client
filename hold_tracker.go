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

package main

import "sync"

// holdTracker macht aus rohen Keydown-/Keyup-Signalen eines Halten-Hotkeys saubere Down/Up-Paare:
// Auto-Wiederholungen beim Halten (z.B. WM_HOTKEY unter Windows) ergeben nur ein Down, und ein Up
// gibt es nur, wenn vorher ein Down gemeldet wurde.
type holdTracker struct {
	mu   sync.Mutex
	held bool
}

// down liefert true, wenn dies der erste Down seit dem letzten Up ist.
func (h *holdTracker) down() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.held {
		return false
	}
	h.held = true
	return true
}

// up liefert true, wenn die Taste zuvor als gedrückt gemeldet war.
func (h *holdTracker) up() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.held {
		return false
	}
	h.held = false
	return true
}

// cancel verwirft einen gemerkten Down, ohne ein Up zu melden (Down wurde nicht ans Frontend
// weitergegeben, z.B. weil das Overlay ausgeblendet war).
func (h *holdTracker) cancel() {
	h.mu.Lock()
	h.held = false
	h.mu.Unlock()
}
