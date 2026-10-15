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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// configPath ist der öffentliche Endpunkt des openDox-Servers mit der Client-Richtlinie
// (Admin-Bereich → Clients). Ohne Login abrufbar, damit sie schon vor der Anmeldung gilt.
const configPath = "/api/v1/settings/client-config/"

// maxConfigSize begrenzt die Antwort; die Richtlinie ist ein paar hundert Bytes groß.
const maxConfigSize = 64 << 10

// Fetch holt die Client-Richtlinie vom Server. Ein Server ohne diesen Endpunkt (ältere
// Version) liefert 404 und damit einen Fehler — dann gelten Cache bzw. settings.json.
func Fetch(ctx context.Context, client *http.Client, serverURL string) (Settings, error) {
	url := strings.TrimRight(serverURL, "/") + configPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Settings{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return Settings{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Settings{}, fmt.Errorf("client-config: HTTP %d", resp.StatusCode)
	}
	var s Settings
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxConfigSize)).Decode(&s); err != nil {
		return Settings{}, fmt.Errorf("client-config: %w", err)
	}
	return s, nil
}

// Merge legt die Server-Richtlinie (remote) über die lokale settings.json: Jedes Feld,
// das der Server setzt, gewinnt — auch eine ausdrücklich leere Allow-Liste. Nur was der
// Server nicht kennt (nil/leer), kommt aus der lokalen Datei.
func Merge(remote, local Settings) Settings {
	merged := local
	if remote.ClipboardExcludeFromHistory != nil {
		merged.ClipboardExcludeFromHistory = remote.ClipboardExcludeFromHistory
	}
	if remote.ClipboardClearAfterSeconds != nil {
		merged.ClipboardClearAfterSeconds = remote.ClipboardClearAfterSeconds
	}
	if remote.InsertAllowlist != nil {
		merged.InsertAllowlist = remote.InsertAllowlist
	}
	if normalize(remote.DictationMicMode) != "" {
		merged.DictationMicMode = remote.DictationMicMode
	}
	if remote.LiveView != nil {
		merged.LiveView = remote.LiveView
	}
	return merged
}

// cacheFile ist der zuletzt vom Server geholte Stand, damit die Richtlinie auch gilt,
// wenn der Server beim Start nicht erreichbar ist. An die Server-URL gebunden: wechselt
// sie, ist der alte Stand wertlos.
type cacheFile struct {
	ServerURL string   `json:"server_url"`
	Settings  Settings `json:"settings"`
}

// LoadCache liest den Cache unter path; ok=false, wenn er fehlt, kaputt ist oder zu
// einem anderen Server gehört.
func LoadCache(path, serverURL string) (Settings, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Settings{}, false
	}
	var c cacheFile
	if err := json.Unmarshal(data, &c); err != nil || c.ServerURL != serverURL {
		return Settings{}, false
	}
	return c.Settings, true
}

// SaveCache schreibt den Cache atomar (temporäre Datei + Rename).
func SaveCache(path, serverURL string, s Settings) error {
	data, err := json.MarshalIndent(cacheFile{ServerURL: serverURL, Settings: s}, "", "  ")
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
