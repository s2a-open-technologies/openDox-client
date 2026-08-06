package main

import (
	"context"
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
	nativeRedirect := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	loginURL := fmt.Sprintf("%s/api/v1/auth/oidc/login?native_redirect=%s",
		a.opendoxURL, url.QueryEscape(nativeRedirect))

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}
	done := make(chan struct{})

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
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
