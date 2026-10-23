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
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestPreferencesWinOverPolicy(t *testing.T) {
	policy := Settings{Theme: "dark", LiveView: ptrBool(true), DictationMicMode: "hold", ClipboardClearAfterSeconds: ptrInt(5)}
	got := Preferences{Theme: "light", LiveView: ptrBool(false), DictationMicMode: "toggle"}.Apply(policy)

	if got.ThemeDefault() != "light" || got.LiveViewDefault() || got.MicHoldMode() {
		t.Errorf("Nutzer-Einstellungen müssen gewinnen: %+v", got)
	}
	if *got.ClipboardClearAfterSeconds != 5 {
		t.Error("Sicherheitswerte dürfen von Preferences nicht berührt werden")
	}
}

func TestEmptyPreferencesKeepPolicy(t *testing.T) {
	policy := Settings{Theme: "light", LiveView: ptrBool(false), DictationMicMode: "toggle"}
	if got := (Preferences{}).Apply(policy); !reflect.DeepEqual(got, policy) {
		t.Errorf("leere Preferences müssen die Vorgabe lassen: %+v", got)
	}
}

func TestPreferencesCleanDropsUnknownValues(t *testing.T) {
	got := Preferences{Theme: "pink", DictationMicMode: " Toggle "}.Clean()
	if got.Theme != "" || got.DictationMicMode != "toggle" {
		t.Errorf("Clean = %+v", got)
	}
}

func TestPreferencesUpdateOnlyTouchesSetFields(t *testing.T) {
	p := Preferences{Theme: "light", DictationMicMode: "toggle"}
	got := p.Update(Preferences{LiveView: ptrBool(false), Theme: "bogus"})
	want := Preferences{Theme: "light", LiveView: ptrBool(false), DictationMicMode: "toggle"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Update = %+v, want %+v", got, want)
	}
}

func TestPreferencesRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "preferences.json")
	want := Preferences{Theme: "light", LiveView: ptrBool(false)}
	if err := SavePreferences(path, want); err != nil {
		t.Fatalf("SavePreferences: %v", err)
	}
	if got := LoadPreferences(path); !reflect.DeepEqual(got, want) {
		t.Errorf("LoadPreferences = %+v, want %+v", got, want)
	}
	if err := os.WriteFile(path, []byte("{kaputt"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := LoadPreferences(path); !reflect.DeepEqual(got, Preferences{}) {
		t.Errorf("kaputte Datei muss leere Preferences ergeben, got %+v", got)
	}
}
