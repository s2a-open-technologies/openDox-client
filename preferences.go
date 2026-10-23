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

package main

import (
	"errors"
	"io/fs"
	"log"
	"os"

	"opendox-client/internal/clientpolicy"
)

// Preferences sind die effektiven Nutzer-Einstellungen fürs Frontend: was der Nutzer im
// Overlay gewählt hat, sonst die Vorgabe aus settings.json bzw. Server-Richtlinie.
type Preferences struct {
	Theme            string `json:"theme"`
	LiveView         bool   `json:"live_view"`
	DictationMicMode string `json:"dictation_mic_mode"`
}

func loadPreferences() clientpolicy.Preferences {
	path, err := preferencesPath()
	if err != nil {
		return clientpolicy.Preferences{}
	}
	return clientpolicy.LoadPreferences(path)
}

// GetPreferences liefert die effektiven Nutzer-Einstellungen.
func (a *App) GetPreferences() Preferences {
	p := a.currentPolicy()
	mode := "hold"
	if !p.MicHoldMode() {
		mode = "toggle"
	}
	return Preferences{Theme: p.ThemeDefault(), LiveView: p.LiveViewDefault(), DictationMicMode: mode}
}

// SavePreferences übernimmt die in u gesetzten Felder in preferences.json; nicht gesetzte
// Felder behalten ihren bisherigen Stand (bzw. weiter die Vorgabe).
func (a *App) SavePreferences(u clientpolicy.Preferences) (Preferences, error) {
	path, err := preferencesPath()
	if err != nil {
		return a.GetPreferences(), err
	}
	a.policyMu.Lock()
	next := a.prefs.Update(u)
	err = clientpolicy.SavePreferences(path, next)
	if err == nil {
		a.prefs = next
	}
	a.policyMu.Unlock()
	if err != nil {
		log.Printf("[opendox] Nutzer-Einstellungen nicht speicherbar: %v", err)
	}
	return a.GetPreferences(), err
}

// ResetPreferences löscht preferences.json — danach gelten wieder die Vorgaben.
func (a *App) ResetPreferences() (Preferences, error) {
	path, err := preferencesPath()
	if err != nil {
		return a.GetPreferences(), err
	}
	a.policyMu.Lock()
	err = os.Remove(path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		err = nil
		a.prefs = clientpolicy.Preferences{}
	}
	a.policyMu.Unlock()
	return a.GetPreferences(), err
}
