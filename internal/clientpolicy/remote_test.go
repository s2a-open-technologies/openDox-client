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
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestMergeRemoteWinsPerField(t *testing.T) {
	local := Settings{
		ClipboardExcludeFromHistory: ptrBool(false),
		ClipboardClearAfterSeconds:  ptrInt(5),
		InsertAllowlist:             []string{"notepad.exe"},
		DictationMicMode:            "toggle",
	}
	remote := Settings{
		ClipboardClearAfterSeconds: ptrInt(60),
		InsertAllowlist:            []string{}, // ausdrücklich leer = Liste aus
		LiveView:                   ptrBool(false),
	}

	got := Merge(remote, local)

	if got.ClipboardExclude() {
		t.Error("ClipboardExclude: nicht vom Server gesetzt, lokaler Wert false muss bleiben")
	}
	if got.ClipboardClearAfter() != 60*time.Second {
		t.Errorf("ClipboardClearAfter = %v, want 60s vom Server", got.ClipboardClearAfter())
	}
	if got.AllowlistActive() {
		t.Error("leere Server-Liste muss die lokale Liste überschreiben")
	}
	if got.MicHoldMode() {
		t.Error("DictationMicMode: nicht vom Server gesetzt, lokales toggle muss bleiben")
	}
	if got.LiveViewDefault() {
		t.Error("LiveView vom Server (false) muss gelten")
	}
}

func TestFetchReadsClientConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/opendox/api/v1/settings/client-config/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"clipboard_exclude_from_history":true,"clipboard_clear_after_seconds":0,` +
			`"insert_allowlist":["medistar.exe"],"dictation_mic_mode":"toggle","live_view":false,"unknown":1}`))
	}))
	defer srv.Close()

	got, err := Fetch(context.Background(), srv.Client(), srv.URL+"/opendox/")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if got.ClipboardClearAfter() != 0 || got.MicHoldMode() || got.LiveViewDefault() || !got.Allows("medistar.exe") || got.Allows("winword.exe") {
		t.Errorf("Fetch lieferte unerwartete Werte: %+v", got)
	}
}

func TestFetchFailsOnOldServerWithoutEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	if _, err := Fetch(context.Background(), srv.Client(), srv.URL); err == nil {
		t.Error("404 muss ein Fehler sein, damit Cache/lokale Datei gelten")
	}
}

func TestCacheRoundTripIsBoundToServerURL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "cache.json")
	want := Settings{ClipboardClearAfterSeconds: ptrInt(10), InsertAllowlist: []string{"a.exe"}}

	if err := SaveCache(path, "https://dox.example", want); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}
	got, ok := LoadCache(path, "https://dox.example")
	if !ok || !reflect.DeepEqual(got, want) {
		t.Errorf("LoadCache = %+v, %v; want %+v, true", got, ok, want)
	}
	if _, ok := LoadCache(path, "https://other.example"); ok {
		t.Error("Cache eines anderen Servers darf nicht gelten")
	}
	if _, ok := LoadCache(filepath.Join(t.TempDir(), "missing.json"), "https://dox.example"); ok {
		t.Error("fehlende Datei darf nicht als Treffer gelten")
	}
}
