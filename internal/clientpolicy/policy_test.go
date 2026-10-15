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
	"testing"
	"time"
)

func ptrBool(b bool) *bool { return &b }
func ptrInt(i int) *int    { return &i }

func TestDefaultsWhenNothingConfigured(t *testing.T) {
	var s Settings
	if !s.ClipboardExclude() {
		t.Error("ClipboardExclude sollte standardmäßig true sein")
	}
	if got := s.ClipboardClearAfter(); got != DefaultClipboardClearAfter {
		t.Errorf("ClipboardClearAfter = %v, want %v", got, DefaultClipboardClearAfter)
	}
	if s.AllowlistActive() {
		t.Error("Allowlist sollte standardmäßig aus sein")
	}
	if !s.Allows("outlook.exe") || !s.Allows("") {
		t.Error("ohne Liste darf alles eingefügt werden")
	}
}

func TestExplicitFalseAndZeroAreRespected(t *testing.T) {
	s := Settings{ClipboardExcludeFromHistory: ptrBool(false), ClipboardClearAfterSeconds: ptrInt(0)}
	if s.ClipboardExclude() {
		t.Error("ausdrückliches false darf nicht durch den Default überschrieben werden")
	}
	if got := s.ClipboardClearAfter(); got != 0 {
		t.Errorf("0 heißt nie leeren, got %v", got)
	}
}

func TestNegativeClearAfterFallsBackToDefault(t *testing.T) {
	s := Settings{ClipboardClearAfterSeconds: ptrInt(-5)}
	if got := s.ClipboardClearAfter(); got != DefaultClipboardClearAfter {
		t.Errorf("got %v, want default", got)
	}
}

func TestCustomClearAfter(t *testing.T) {
	s := Settings{ClipboardClearAfterSeconds: ptrInt(90)}
	if got := s.ClipboardClearAfter(); got != 90*time.Second {
		t.Errorf("got %v", got)
	}
}

func TestAllowlistMatching(t *testing.T) {
	s := Settings{InsertAllowlist: []string{" Notepad.EXE ", "orbis", ""}}
	if !s.AllowlistActive() {
		t.Fatal("Liste sollte aktiv sein")
	}
	for _, ok := range []string{"notepad.exe", "NOTEPAD.EXE", "orbis.exe", "orbis"} {
		if !s.Allows(ok) {
			t.Errorf("%q sollte erlaubt sein", ok)
		}
	}
	for _, no := range []string{"outlook.exe", "", "orbis2.exe", "notepad.exe.bak"} {
		if s.Allows(no) {
			t.Errorf("%q sollte nicht erlaubt sein", no)
		}
	}
}

func TestListWithOnlyBlankEntriesIsInactive(t *testing.T) {
	s := Settings{InsertAllowlist: []string{"", "  "}}
	if s.AllowlistActive() || !s.Allows("anything.exe") {
		t.Error("nur leere Einträge dürfen die Liste nicht aktivieren")
	}
}

func TestSettingsJSONRoundTrip(t *testing.T) {
	var s Settings
	raw := `{"clipboard_exclude_from_history": false, "clipboard_clear_after_seconds": 0, "insert_allowlist": ["a.exe"]}`
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatal(err)
	}
	if s.ClipboardExclude() || s.ClipboardClearAfter() != 0 || !s.Allows("a.exe") || s.Allows("b.exe") {
		t.Errorf("JSON nicht korrekt übernommen: %+v", s)
	}
	var empty Settings
	if err := json.Unmarshal([]byte(`{}`), &empty); err != nil {
		t.Fatal(err)
	}
	if !empty.ClipboardExclude() || empty.ClipboardClearAfter() != DefaultClipboardClearAfter {
		t.Error("fehlende Schlüssel müssen die Defaults ergeben")
	}
}

func TestLiveViewDefault(t *testing.T) {
	off, on := false, true
	if !(Settings{}).LiveViewDefault() || !(Settings{LiveView: &on}).LiveViewDefault() {
		t.Error("Default und true müssen die Live-Ansicht einschalten")
	}
	if (Settings{LiveView: &off}).LiveViewDefault() {
		t.Error("live_view=false wurde nicht übernommen")
	}
}

func TestMicHoldMode(t *testing.T) {
	for mode, want := range map[string]bool{"": true, "hold": true, "HOLD": true, "unbekannt": true, "toggle": false, " Toggle ": false} {
		if got := (Settings{DictationMicMode: mode}).MicHoldMode(); got != want {
			t.Errorf("MicHoldMode(%q) = %v, want %v", mode, got, want)
		}
	}
}
