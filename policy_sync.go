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
	"reflect"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"opendox-client/internal/clientpolicy"
)

// Die Client-Richtlinie (Zwischenablage, Allow-Liste, Mic-Modus, Live-Ansicht) verwaltet der
// Admin im openDox-Server (Admin → Clients). Der Client holt sie beim Start und danach
// regelmäßig; Server-Werte gewinnen gegenüber settings.json (clientpolicy.Merge). Der letzte
// Stand wird zwischengespeichert, damit er auch gilt, wenn der Server beim Start fehlt.
const (
	policyRefreshInterval = 10 * time.Minute
	policyFetchTimeout    = 10 * time.Second
	policyCacheFileName   = "client-policy-cache.json"

	// eventPolicyChanged: die Richtlinie hat sich zur Laufzeit geändert — das Frontend
	// holt dann z.B. den Live-View-Default neu (GetLiveViewDefault).
	eventPolicyChanged = "policy:changed"
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

// currentPolicy liefert die gerade gültige Richtlinie.
func (a *App) currentPolicy() clientpolicy.Settings {
	a.policyMu.RLock()
	defer a.policyMu.RUnlock()
	return a.policy
}

// syncPolicyLoop holt die Richtlinie sofort und danach alle policyRefreshInterval.
func (a *App) syncPolicyLoop() {
	a.refreshPolicy()
	ticker := time.NewTicker(policyRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.refreshPolicy()
		}
	}
}

// refreshPolicy holt die Richtlinie vom Server. Bei einem Fehler (Server weg, ältere
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
	changed := !reflect.DeepEqual(a.policy, merged)
	a.policy = merged
	a.policyMu.Unlock()
	if changed {
		log.Printf("[opendox] Client-Richtlinie vom Server übernommen")
		runtime.EventsEmit(a.ctx, eventPolicyChanged)
	}
}
