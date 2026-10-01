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

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"opendox-client/internal/clientpolicy"
)

func writeSettings(t *testing.T, content string) func() (string, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return func() (string, error) { return path, nil }
}

// Die Verhaltens-Einstellungen stehen flach neben server_url in derselben Datei.
func TestTrySettingsFileReadsPolicyKeysFlat(t *testing.T) {
	s, ok := trySettingsFile(writeSettings(t, `{
		"server_url": "https://opendox.example",
		"clipboard_exclude_from_history": false,
		"clipboard_clear_after_seconds": 90,
		"insert_allowlist": ["Orbis.exe"],
		"dictation_mic_mode": "toggle",
		"live_view": false
	}`))
	if !ok {
		t.Fatal("settings.json wurde nicht akzeptiert")
	}
	if s.ServerURL != "https://opendox.example" {
		t.Errorf("ServerURL = %q", s.ServerURL)
	}
	var p clientpolicy.Settings = s.policySettings
	if p.ClipboardExclude() {
		t.Error("clipboard_exclude_from_history: false wurde nicht übernommen")
	}
	if got := p.ClipboardClearAfter(); got != 90*time.Second {
		t.Errorf("ClipboardClearAfter = %v", got)
	}
	if p.MicHoldMode() || p.LiveViewDefault() {
		t.Error("dictation_mic_mode/live_view wurden nicht übernommen")
	}
	if !p.Allows("orbis.exe") || p.Allows("outlook.exe") {
		t.Error("insert_allowlist wurde nicht übernommen")
	}
}

// Eine settings.json nur mit server_url (bisheriges Format) ergibt die Defaults.
func TestTrySettingsFileLegacyFileGivesPolicyDefaults(t *testing.T) {
	s, ok := trySettingsFile(writeSettings(t, `{"server_url": "http://localhost:10000"}`))
	if !ok {
		t.Fatal("altes Format muss weiter funktionieren")
	}
	p := s.policySettings
	if !p.ClipboardExclude() || p.ClipboardClearAfter() != clientpolicy.DefaultClipboardClearAfter || p.AllowlistActive() {
		t.Errorf("Defaults nicht korrekt: %+v", p)
	}
}

func TestTrySettingsFileStillRequiresServerURL(t *testing.T) {
	if _, ok := trySettingsFile(writeSettings(t, `{"insert_allowlist": ["a.exe"]}`)); ok {
		t.Error("ohne server_url muss die Datei weiter abgelehnt werden")
	}
}

func TestSettingsDirForExe(t *testing.T) {
	sep := string(filepath.Separator)
	cases := map[string]string{
		filepath.Join(sep+"opt", "opendox", "opendox-client"):                                   filepath.Join(sep+"opt", "opendox"),
		filepath.Join(sep+"Applications", "openDox.app", "Contents", "MacOS", "opendox-client"): sep + "Applications",
	}
	for exe, want := range cases {
		if got := settingsDirForExe(exe); got != want {
			t.Errorf("settingsDirForExe(%q) = %q, want %q", exe, got, want)
		}
	}
}
