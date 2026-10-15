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

// Package clientpolicy hält die plattformunabhängige Logik für die
// Datenschutz-Einstellungen des Clients (Zwischenablage, Einfügeziel), damit sie
// ohne Wails/robotgo/X11 testbar ist.
package clientpolicy

import (
	"strings"
	"time"
)

// DefaultClipboardClearAfter ist die Frist, nach der der Client die Zwischenablage
// nach dem Einfügen leert, wenn settings.json nichts anderes vorgibt.
const DefaultClipboardClearAfter = 30 * time.Second

// Settings sind die optionalen Verhaltens-Einstellungen aus settings.json. Zeiger
// bzw. leere Slices bedeuten "nicht gesetzt" (dann gilt der Default) — so lässt sich
// ein ausdrückliches false/0 von einem fehlenden Schlüssel unterscheiden.
type Settings struct {
	// ClipboardExcludeFromHistory: Text als "nicht in Verlauf/Cloud aufnehmen"
	// markieren (Windows). Default true.
	ClipboardExcludeFromHistory *bool `json:"clipboard_exclude_from_history"`
	// ClipboardClearAfterSeconds: Zwischenablage so viele Sekunden nach dem Einfügen
	// leeren, sofern unverändert. 0 = nie leeren. Default 30.
	ClipboardClearAfterSeconds *int `json:"clipboard_clear_after_seconds"`
	// InsertAllowlist: .exe-Namen der Programme, in die ohne Rückfrage eingefügt
	// werden darf. Leer = Funktion aus (Default) — der Client ist ein allgemeines
	// Diktierwerkzeug, die Liste ist nur eine Option für IT-verwaltete Arbeitsplätze.
	InsertAllowlist []string `json:"insert_allowlist"`
	// DictationMicMode: Verhalten der Record-Taste eines Tastenmikrofons (SpeechMike/PowerMic).
	// "hold" (Default): Taste halten = aufnehmen, loslassen = stoppen und Text automatisch
	// einfügen. "toggle": bisheriges Verhalten (Druck startet, nächster Druck stoppt).
	DictationMicMode string `json:"dictation_mic_mode"`
	// LiveView: Default für die Live-Ansicht (Text erscheint während des Sprechens). Kann der
	// Nutzer im Overlay pro Rechner überschreiben. Default true.
	LiveView *bool `json:"live_view"`
}

// LiveViewDefault liefert den Default der Live-Ansicht; nicht gesetzt = an.
func (s Settings) LiveViewDefault() bool {
	return s.LiveView == nil || *s.LiveView
}

// MicHoldMode meldet, ob die Record-Taste des Tastenmikrofons als Push-to-Talk arbeitet.
// Alles außer "toggle" (auch leer/unbekannt) ergibt den Default "hold".
func (s Settings) MicHoldMode() bool {
	return normalize(s.DictationMicMode) != "toggle"
}

// ClipboardExclude meldet, ob die Zwischenablage als vertraulich markiert werden soll.
func (s Settings) ClipboardExclude() bool {
	return s.ClipboardExcludeFromHistory == nil || *s.ClipboardExcludeFromHistory
}

// ClipboardClearAfter liefert die Frist bis zum Leeren; 0 heißt "nie". Fehlende oder
// negative Werte ergeben den Default.
func (s Settings) ClipboardClearAfter() time.Duration {
	if s.ClipboardClearAfterSeconds == nil || *s.ClipboardClearAfterSeconds < 0 {
		return DefaultClipboardClearAfter
	}
	return time.Duration(*s.ClipboardClearAfterSeconds) * time.Second
}

// AllowlistActive ist true, sobald mindestens ein nichtleerer Eintrag konfiguriert ist.
func (s Settings) AllowlistActive() bool {
	for _, entry := range s.InsertAllowlist {
		if normalize(entry) != "" {
			return true
		}
	}
	return false
}

// Allows meldet, ob in das Programm process (.exe-Name) ohne Rückfrage eingefügt
// werden darf. Ohne aktive Liste immer true; bei aktiver Liste ist ein unbekanntes
// Programm (leerer Name) nicht erlaubt. Der Vergleich ignoriert Groß-/Kleinschreibung,
// und ein Eintrag ohne ".exe" trifft ebenfalls ("notepad" = "notepad.exe").
func (s Settings) Allows(process string) bool {
	if !s.AllowlistActive() {
		return true
	}
	p := normalize(process)
	if p == "" {
		return false
	}
	for _, entry := range s.InsertAllowlist {
		e := normalize(entry)
		if e != "" && (e == p || e+".exe" == p) {
			return true
		}
	}
	return false
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
