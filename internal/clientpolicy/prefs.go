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

package clientpolicy

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Preferences sind die Einstellungen, die der Nutzer selbst im Overlay umschaltet
// (preferences.json im Benutzer-Konfigordner). Sie gewinnen gegenüber settings.json und
// Server-Richtlinie, die dafür nur die Vorgabe liefern. Leer/nil = nicht gewählt, dann gilt
// die Vorgabe. Sicherheitswerte (Zwischenablage, Allow-Liste) gehören bewusst nicht hierher.
type Preferences struct {
	Theme            string `json:"theme,omitempty"`
	LiveView         *bool  `json:"live_view,omitempty"`
	DictationMicMode string `json:"dictation_mic_mode,omitempty"`
}

// Clean verwirft unbekannte Werte, damit eine von Hand kaputt editierte Datei nur auf die
// Vorgabe zurückfällt statt ein undefiniertes Verhalten zu erzeugen.
func (p Preferences) Clean() Preferences {
	switch normalize(p.Theme) {
	case "dark", "light":
		p.Theme = normalize(p.Theme)
	default:
		p.Theme = ""
	}
	switch normalize(p.DictationMicMode) {
	case "hold", "toggle":
		p.DictationMicMode = normalize(p.DictationMicMode)
	default:
		p.DictationMicMode = ""
	}
	return p
}

// Update übernimmt die in u gesetzten Felder (nur gültige Werte), der Rest bleibt.
func (p Preferences) Update(u Preferences) Preferences {
	u = u.Clean()
	if u.Theme != "" {
		p.Theme = u.Theme
	}
	if u.LiveView != nil {
		p.LiveView = u.LiveView
	}
	if u.DictationMicMode != "" {
		p.DictationMicMode = u.DictationMicMode
	}
	return p
}

// Apply legt die Nutzer-Einstellungen über die Richtlinie s.
func (p Preferences) Apply(s Settings) Settings {
	p = p.Clean()
	if p.Theme != "" {
		s.Theme = p.Theme
	}
	if p.LiveView != nil {
		s.LiveView = p.LiveView
	}
	if p.DictationMicMode != "" {
		s.DictationMicMode = p.DictationMicMode
	}
	return s
}

// LoadPreferences liest die Datei unter path; fehlt sie oder ist sie kaputt, gibt es
// leere Preferences (= überall die Vorgabe).
func LoadPreferences(path string) Preferences {
	data, err := os.ReadFile(path)
	if err != nil {
		return Preferences{}
	}
	var p Preferences
	if err := json.Unmarshal(data, &p); err != nil {
		return Preferences{}
	}
	return p.Clean()
}

// SavePreferences schreibt die Datei atomar (temporäre Datei + Rename).
func SavePreferences(path string, p Preferences) error {
	data, err := json.MarshalIndent(p.Clean(), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
