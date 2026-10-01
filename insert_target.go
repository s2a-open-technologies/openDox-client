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

package main

// InsertTarget beschreibt das Programm, in das ein Einfügen gerade ginge. Wird vom
// Overlay angezeigt, damit ein Diktat nicht unbemerkt im falschen Fenster landet.
type InsertTarget struct {
	// Process ist der .exe-Name in Kleinbuchstaben; leer, wenn unbekannt oder nicht
	// unterstützt (macOS/Linux) — das Frontend blendet die Anzeige dann aus.
	Process string `json:"process"`
	// Title ist der Fenstertitel. Kann Patientennamen enthalten: nur anzeigen, nie
	// loggen oder speichern.
	Title string `json:"title"`
	// ListActive ist true, wenn eine Allow-Liste konfiguriert ist (Server oder settings.json).
	ListActive bool `json:"list_active"`
	// Allowed ist false, wenn eine Liste aktiv ist und Process nicht darin steht —
	// dann fragt das Frontend vor dem Einfügen nach.
	Allowed bool `json:"allowed"`
}

// GetInsertTarget liefert das aktuelle Einfügeziel und ob es laut Richtlinie ohne
// Rückfrage erlaubt ist.
func (a *App) GetInsertTarget() InsertTarget {
	process, title := foregroundTarget()
	policy := a.currentPolicy()
	return InsertTarget{
		Process:    process,
		Title:      title,
		ListActive: policy.AllowlistActive(),
		Allowed:    policy.Allows(process),
	}
}
