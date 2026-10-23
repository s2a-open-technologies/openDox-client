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
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"opendox-client/internal/clientpolicy"
)

// Die Vorgaben für den Client (Zwischenablage, Allow-Liste, Mic-Modus, Live-Ansicht) kann der
// Admin im openDox-Server pflegen (Admin → Clients). Der Client holt sie einmal beim Start;
// was settings.json am Rechner festlegt, gewinnt (clientpolicy.Merge), und darüber liegen die
// Nutzer-Einstellungen aus preferences.json. Der letzte Server-Stand wird zwischengespeichert,
// damit er auch gilt, wenn der Server beim Start fehlt.
const (
	policyFetchTimeout  = 10 * time.Second
	policyCacheFileName = "client-policy-cache.json"
)

// policyCachePath liegt im plattformüblichen Cache-Ordner des Benutzers
// (~/.cache, ~/Library/Caches, %LocalAppData%).
func policyCachePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appConfigDirName, policyCacheFileName), nil
}

// initialPolicy ist die Richtlinie beim Programmstart: zwischengespeicherter Server-Stand
// über settings.json, ohne auf das Netz zu warten.
func initialPolicy(serverURL string, local clientpolicy.Settings) clientpolicy.Settings {
	path, err := policyCachePath()
	if err != nil {
		return local
	}
	cached, ok := clientpolicy.LoadCache(path, serverURL)
	if !ok {
		return local
	}
	return clientpolicy.Merge(cached, local)
}

// currentPolicy liefert die gerade gültige Richtlinie, mit den Nutzer-Einstellungen darüber.
func (a *App) currentPolicy() clientpolicy.Settings {
	a.policyMu.RLock()
	defer a.policyMu.RUnlock()
	return a.prefs.Apply(a.policy)
}

// refreshPolicy holt die Richtlinie einmal beim Start vom Server. Bei einem Fehler (Server weg, ältere
// Server-Version ohne Endpunkt) bleibt der bisherige Stand unverändert.
func (a *App) refreshPolicy() {
	ctx, cancel := context.WithTimeout(a.ctx, policyFetchTimeout)
	defer cancel()
	remote, err := clientpolicy.Fetch(ctx, http.DefaultClient, a.opendoxURL)
	if err != nil {
		log.Printf("[opendox] Client-Richtlinie nicht abrufbar, nutze bisherigen Stand: %v", err)
		return
	}
	if path, err := policyCachePath(); err == nil {
		if err := clientpolicy.SaveCache(path, a.opendoxURL, remote); err != nil {
			log.Printf("[opendox] Client-Richtlinie: Cache nicht schreibbar: %v", err)
		}
	}

	merged := clientpolicy.Merge(remote, a.localPolicy)
	a.policyMu.Lock()
	a.policy = merged
	a.policyMu.Unlock()
	log.Printf("[opendox] Client-Richtlinie vom Server übernommen")
}
