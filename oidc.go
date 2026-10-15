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
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var oidcLoginInFlight atomic.Bool

const oidcCallbackSuccessHTML = `<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding-top:4rem;">
<h2>Login erfolgreich</h2><p>Sie können dieses Fenster jetzt schließen und zu openDox zurückkehren.</p>
</body></html>`

const oidcCallbackErrorHTML = `<!DOCTYPE html><html><body style="font-family:sans-serif;text-align:center;padding-top:4rem;">
<h2>Login fehlgeschlagen</h2><p>Bitte schließen Sie dieses Fenster und versuchen Sie es erneut.</p>
</body></html>`

// LoginWithOIDC startet den nativen OIDC-Login-Flow (RFC 8252 Loopback-Redirect): öffnet
// den System-Browser auf die OIDC-Login-URL des Servers, wartet auf den Callback an einem
// lokalen Loopback-Listener und meldet das Ergebnis per Wails-Event an das Frontend
// ("oidc:login-complete" mit access_token/refresh_token, oder "oidc:login-error").
//
// Kehrt zurück, sobald der Listener steht und der Browser geöffnet wurde - wartet NICHT
// auf den Abschluss der eigentlichen Anmeldung im Browser.
func (a *App) LoginWithOIDC() error {
	if !oidcLoginInFlight.CompareAndSwap(false, true) {
		return fmt.Errorf("OIDC-Login läuft bereits")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0") // Loopback only, OS wählt freien Port
	if err != nil {
		oidcLoginInFlight.Store(false)
		return fmt.Errorf("konnte lokalen Listener nicht starten: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	// Der Callback-Pfad enthält ein pro Login-Versuch zufälliges Token. Der Server MUSS
	// exakt an native_redirect zurückleiten, damit der Loopback-Flow überhaupt funktioniert
	// - ein beliebiger anderer lokaler Prozess/Tab, der /callback erraten würde, kennt das
	// Token nicht und kann so keine Tokens einschleusen (Schutz gegen unauthentifizierte
	// Callback-Aufrufe).
	callbackToken, err := randomToken()
	if err != nil {
		oidcLoginInFlight.Store(false)
		_ = listener.Close()
		return fmt.Errorf("konnte kein Zufalls-Token erzeugen: %w", err)
	}
	callbackPath := "/callback/" + callbackToken
	nativeRedirect := fmt.Sprintf("http://127.0.0.1:%d%s", port, callbackPath)
	loginURL := fmt.Sprintf("%s/api/v1/auth/oidc/login?native_redirect=%s",
		a.opendoxURL, url.QueryEscape(nativeRedirect))

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}
	done := make(chan struct{})

	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		accessToken := r.URL.Query().Get("access_token")
		refreshToken := r.URL.Query().Get("refresh_token")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if accessToken == "" || refreshToken == "" {
			fmt.Fprint(w, oidcCallbackErrorHTML)
			runtime.EventsEmit(a.ctx, "oidc:login-error", "Kein Token vom Server erhalten")
		} else {
			fmt.Fprint(w, oidcCallbackSuccessHTML)
			runtime.EventsEmit(a.ctx, "oidc:login-complete", accessToken, refreshToken)
		}
		close(done)
	})

	go func() {
		_ = srv.Serve(listener)
	}()

	go func() {
		select {
		case <-done:
		case <-time.After(5 * time.Minute):
			runtime.EventsEmit(a.ctx, "oidc:login-error", "Zeitüberschreitung beim Login")
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		oidcLoginInFlight.Store(false)
	}()

	runtime.BrowserOpenURL(a.ctx, loginURL)
	return nil
}

// randomToken erzeugt ein kryptografisch zufälliges, URL-sicheres Token für den
// Callback-Pfad (siehe LoginWithOIDC).
func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
