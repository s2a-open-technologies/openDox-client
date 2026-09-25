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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.design/x/hotkey"
)

// --- OpenDox API Typen (passen zur OpenAPI-Spec) ---

type RecordCreate struct {
	UserID     int    `json:"user_id"`
	Title      string `json:"title"`
	CategoryID *int   `json:"category_id"`
}

type Record struct {
	SessionID string `json:"session_id"`
	ID        int    `json:"id"`
}

type ActiveSession struct {
	SessionID string `json:"session_id"`
}

type WhisperSettings struct {
	LiveBackend string `json:"live_backend"`
}

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type TaskStatus struct {
	TaskID int    `json:"task_id"`
	Status string `json:"status"`
	Result string `json:"result"`
	Error  string `json:"error"`
}

type RecordDetails struct {
	ID        int    `json:"id"`
	TaskID    int    `json:"task_id"`
	Summarize string `json:"summarize"`
}

type Settings struct {
	ServerURL string `json:"server_url"`
}

// appConfigDirName ist der Ordnername für die Laufzeit-Konfig, siehe runtimeSettingsPath.
const appConfigDirName = "opendox-client"

// runtimeSettingsPath liegt außerhalb des .app-Bundles, an einem plattformüblichen
// Konfigurationsort (os.UserConfigDir(): ~/Library/Application Support auf macOS, %AppData%
// auf Windows, ~/.config bzw. $XDG_CONFIG_HOME auf Linux). Damit lässt sich pro Kunde ein
// anderer Server eintragen, ohne die App neu zu bauen — einfach diese Datei bei der
// Installation dort ablegen.
func runtimeSettingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appConfigDirName, "settings.json"), nil
}

// loadSettings liest die Laufzeit-Konfigdatei (siehe runtimeSettingsPath). Existiert sie nicht
// oder lässt sie sich nicht parsen (z.B. lokale Entwicklung ohne Installation), fällt es auf den
// hartkodierten Default zurück — es gibt bewusst keine zur Compile-Zeit eingebettete
// settings.json mehr, damit dieselbe Build für jeden Kunden funktioniert.
func loadSettings() Settings {
	path, err := runtimeSettingsPath()
	if err != nil {
		log.Printf("[opendox] Laufzeit-Konfig-Pfad nicht ermittelbar: %v — nutze Default", err)
		return Settings{ServerURL: "http://localhost:10000"}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[opendox] keine Laufzeit-Konfig unter %s (%v) — nutze Default", path, err)
		return Settings{ServerURL: "http://localhost:10000"}
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil || s.ServerURL == "" {
		log.Printf("[opendox] Laufzeit-Konfig %s ungültig (%v) — nutze Default", path, err)
		return Settings{ServerURL: "http://localhost:10000"}
	}

	log.Printf("[opendox] Server-URL aus Laufzeit-Konfig geladen: %s", path)
	return s
}

func deriveWSURL(serverURL string) string {
	if strings.HasPrefix(serverURL, "https://") {
		return "wss://" + serverURL[8:]
	}
	return "ws://" + serverURL[7:]
}

// --- App Struct ---

type App struct {
	ctx        context.Context
	opendoxURL string
	wsURL      string

	mu      sync.Mutex
	visible bool

	// hotkeyFailures zählt die Hotkeys, deren Registrierung fehlgeschlagen ist
	// (typischerweise weil macOS-Eingabeüberwachung fehlt). Wird über
	// CheckHotkeys() ans Frontend gemeldet, damit der Nutzer einen Hinweis
	// bekommt statt sich über nicht funktionierende Tastenkürzel zu wundern.
	hotkeyFailures int

	// insertMu serialisiert InsertTranscript-Aufrufe. Ohne das liefen bei
	// mehrfachem Drücken von Ctrl+Alt+V (Debounce ist nur 700ms, ein Insert
	// inkl. Sleep+Tippen kann aber länger dauern) mehrere robotgo.TypeStr-Aufrufe
	// gleichzeitig - die interleaven ihre Tastatur-Injektion zu verstümmeltem
	// Text ("Ja, wo ist denn ??????" statt des vollständigen Satzes).
	insertMu sync.Mutex
}

func NewApp() *App {
	s := loadSettings()
	return &App{
		opendoxURL: s.ServerURL,
		wsURL:      deriveWSURL(s.ServerURL),
		visible:    true,
	}
}

// GetSettings liefert die aktuellen Server-Einstellungen an das Frontend.
func (a *App) GetSettings() Settings {
	return Settings{ServerURL: a.opendoxURL}
}

// --- Lifecycle ---

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Auf macOS: Accessibility-Berechtigung prüfen. AXIsProcessTrusted() kann
	// auf neuerem macOS langsam sein (TCC-Abfrage) — mit Timeout absichern.
	accDone := make(chan bool, 1)
	go func() {
		accDone <- ensureAccessibility()
	}()
	select {
	case acc := <-accDone:
		if !acc {
			log.Printf("[opendox] Accessibility-Berechtigung fehlt — Hotkeys und Einfügen deaktiviert bis in den Systemeinstellungen erlaubt")
		}
	case <-time.After(5 * time.Second):
		log.Printf("[opendox] Accessibility-Prüfung abgebrochen nach 5s")
	}
	a.registerGlobalHotkeys()
	a.startDictationMic()
}

// Event-Namen, die sowohl von den Tastatur-Hotkeys als auch von den physischen
// Diktiermikro-Tasten (dictation_mic.go) emittiert werden — Frontend kennt nur diese drei.
const (
	eventToggleRecord = "hotkey:toggle-record"
	eventTogglePause  = "hotkey:toggle-pause"
	eventAcceptInsert = "hotkey:accept-insert"
)

// registerGlobalHotkeys registriert die systemweiten Hotkeys (funktionieren unabhängig
// davon, welches Fenster gerade fokussiert ist) und meldet jeden Tastendruck als
// Wails-Event an das Frontend.
func (a *App) registerGlobalHotkeys() {
	// golang.design/x/hotkey nutzt CGEventTap — auf macOS kann Register() blockieren,
	// wenn TCC-Berechtigungen langsam abgefragt werden oder das EventTap-Setup hängt.
	// Daher mit Timeout absichern.
	type regJob struct {
		mods       []hotkey.Modifier
		key        hotkey.Key
		eventName  string
		isWindowHK bool
	}
	jobs := []regJob{
		{hotkeyMods, hotkey.KeyR, eventToggleRecord, false},
		{hotkeyMods, hotkey.KeyP, eventTogglePause, false},
		{hotkeyMods, hotkey.KeyV, eventAcceptInsert, false},
		{hotkeyMods, hotkey.KeyH, "", true},
	}

	done := make(chan struct{})
	go func() {
		for _, j := range jobs {
			if j.isWindowHK {
				a.registerWindowHotkeyTimeout(j.mods, j.key)
			} else {
				a.registerHotkeyTimeout(j.mods, j.key, j.eventName)
			}
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		log.Printf("[opendox] hotkey registration timed out after 10s")
	}
}

// CheckAccessibility meldet dem Frontend, ob der Prozess für Accessibility
// (Bedienhilfen) vertraut ist — nur dann funktionieren auf macOS Tastatur-
// Injektion (robotgo.TypeStr/Paste). Auf Linux/Windows liefert der Stub
// (accessibility_stub.go) immer true.
func (a *App) CheckAccessibility() bool {
	return ensureAccessibility()
}

// CheckHotkeys meldet dem Frontend, ob alle systemweiten Hotkeys erfolgreich
// registriert wurden. Auf macOS benötigt CGEventTap (golang.design/x/hotkey)
// die separate Berechtigung "Eingabeüberwachung" (Input Monitoring) — selbst
// wenn Bedienhilfen aktiv sind. Liefert true, wenn keine Fehler auftraten.
func (a *App) CheckHotkeys() bool {
	return a.hotkeyFailures == 0
}

// registerHotkey registriert eine einzelne Tastenkombination und emittiert bei jedem
// (entprellten) Tastendruck das angegebene Wails-Event.
func (a *App) registerHotkey(mods []hotkey.Modifier, key hotkey.Key, eventName string) {
	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		log.Printf("[opendox] failed to register global hotkey for %s: %v", eventName, err)
		a.hotkeyFailures++
		return
	}
	go func() {
		var last time.Time
		for range hk.Keydown() {
			if time.Since(last) < 700*time.Millisecond {
				continue // debounce X11 auto-repeat while die Kombination gehalten wird
			}
			last = time.Now()
			a.emitHotkeyEvent(eventName)
		}
	}()
}

// registerHotkeyTimeout ist die timeout-geschützte Version von registerHotkey.
func (a *App) registerHotkeyTimeout(mods []hotkey.Modifier, key hotkey.Key, eventName string) {
	type regResult struct {
		err error
		hk  *hotkey.Hotkey
	}
	ch := make(chan regResult, 1)
	go func() {
		hk := hotkey.New(mods, key)
		err := hk.Register()
		ch <- regResult{err, hk}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			log.Printf("[opendox] failed to register global hotkey for %s: %v", eventName, res.err)
			a.hotkeyFailures++
			return
		}
		go func() {
			var last time.Time
			for range res.hk.Keydown() {
				if time.Since(last) < 700*time.Millisecond {
					continue
				}
				last = time.Now()
				a.emitHotkeyEvent(eventName)
			}
		}()
	case <-time.After(10 * time.Second):
		log.Printf("[opendox] hotkey register timed out for %s", eventName)
		a.hotkeyFailures++
	}
}

// registerWindowHotkeyTimeout ist die timeout-geschützte Version von registerWindowHotkey.
func (a *App) registerWindowHotkeyTimeout(mods []hotkey.Modifier, key hotkey.Key) {
	type regResult struct {
		err error
		hk  *hotkey.Hotkey
	}
	ch := make(chan regResult, 1)
	go func() {
		hk := hotkey.New(mods, key)
		err := hk.Register()
		ch <- regResult{err, hk}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			log.Printf("[opendox] failed to register window hotkey: %v", res.err)
			return
		}
		go func() {
			var last time.Time
			for range res.hk.Keydown() {
				if time.Since(last) < 700*time.Millisecond {
					continue
				}
				last = time.Now()
				a.mu.Lock()
				if a.visible {
					runtime.WindowHide(a.ctx)
					a.visible = false
				} else {
					runtime.WindowShow(a.ctx)
					a.positionTopRight()
					a.visible = true
				}
				a.mu.Unlock()
			}
		}()
	case <-time.After(10 * time.Second):
		log.Printf("[opendox] window hotkey register timed out")
	}
}

// emitHotkeyEvent meldet eines der drei Hotkey-Events ans Frontend, sofern das Overlay nicht
// per Ctrl+Alt+H ausgeblendet ist. Gemeinsamer Pfad für Tastatur-Hotkeys (oben) und physische
// Diktiermikro-Tasten (dictation_mic.go) — beide sollen sich bei ausgeblendetem Overlay gleich
// verhalten.
func (a *App) emitHotkeyEvent(eventName string) {
	a.mu.Lock()
	visible := a.visible
	a.mu.Unlock()
	if !visible {
		return
	}
	runtime.EventsEmit(a.ctx, eventName)
}

// registerWindowHotkey registriert eine Tastenkombination (Ctrl+Alt+H) zum Ein- und
// Ausblenden des Overlays. Läuft komplett in Go, da es eine reine Fensteroperation ist.
func (a *App) registerWindowHotkey(mods []hotkey.Modifier, key hotkey.Key) {
	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		log.Printf("[opendox] failed to register window hotkey: %v", err)
		return
	}
	go func() {
		var last time.Time
		for range hk.Keydown() {
			if time.Since(last) < 700*time.Millisecond {
				continue // debounce X11 auto-repeat
			}
			last = time.Now()
			a.mu.Lock()
			if a.visible {
				runtime.WindowHide(a.ctx)
				a.visible = false
			} else {
				runtime.WindowShow(a.ctx)
				a.positionTopRight()
				a.visible = true
			}
			a.mu.Unlock()
		}
	}()
}

func (a *App) domReady(ctx context.Context) {
	a.ctx = ctx
	setupWebviewPermissions()
	a.positionTopRight()
}

func (a *App) positionTopRight() {
	const margin = 16
	w, _ := runtime.WindowGetSize(a.ctx)
	screens, err := runtime.ScreenGetAll(a.ctx)
	if err != nil {
		log.Printf("[opendox] could not get screens: %v", err)
		return
	}
	screenW := 0
	for _, s := range screens {
		if s.IsPrimary {
			screenW = s.Size.Width
		}
	}
	if screenW == 0 && len(screens) > 0 {
		screenW = screens[0].Size.Width
	}
	x := screenW - w - margin
	if x < 0 {
		x = 0
	}
	runtime.WindowSetPosition(a.ctx, x, margin)
}

// --- Hilfsfunktion: HTTP Request mit Bearer-Token ---

func (a *App) doRequest(method, path, token string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, a.opendoxURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return http.DefaultClient.Do(req)
}

// --- Bound Methods (werden als TS-Funktionen generiert) ---

// CreateRecord erstellt eine neue Aufnahme beim OpenDox-Server.
// categoryID 0 bedeutet "keine Kategorie" (wird als null gesendet).
func (a *App) CreateRecord(userID int, title string, categoryID int, token string) (Record, error) {
	req := RecordCreate{
		UserID: userID,
		Title:  title,
	}
	if categoryID > 0 {
		req.CategoryID = &categoryID
	}
	resp, err := a.doRequest("POST", "/api/v1/records/create", token, req)
	if err != nil {
		return Record{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return Record{}, fmt.Errorf("create record failed: %s", string(b))
	}

	var r Record
	json.NewDecoder(resp.Body).Decode(&r)
	return r, nil
}

// GetActiveSession fragt ab, ob es eine unfertige Aufnahmesession gibt.
func (a *App) GetActiveSession(userID int, token string) (ActiveSession, error) {
	resp, err := a.doRequest("GET", fmt.Sprintf("/api/v1/records/active-session/%d", userID), token, nil)
	if err != nil {
		return ActiveSession{}, err
	}
	defer resp.Body.Close()

	var s ActiveSession
	json.NewDecoder(resp.Body).Decode(&s)
	return s, nil
}

// CancelSession bricht eine unfertige Aufnahmesession ab.
func (a *App) CancelSession(userID int, token string) error {
	resp, err := a.doRequest("DELETE", fmt.Sprintf("/api/v1/records/cancel-session/%d", userID), token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// GetWSURL baut die WebSocket-URL für die Live-Transkription zusammen.
// Token kommt aus dem Frontend (localStorage).
func (a *App) GetWSURL(sessionID, userID, token string) string {
	return fmt.Sprintf("%s/api/v1/records/ws/record?session_id=%s&user_id=%s&token=%s",
		a.wsURL,
		url.QueryEscape(sessionID),
		url.QueryEscape(userID),
		url.QueryEscape(token),
	)
}

// GetWhisperSettings liefert die Whisper/ASR-Konfiguration (öffentlich, kein Token nötig).
func (a *App) GetWhisperSettings() (WhisperSettings, error) {
	resp, err := a.doRequest("GET", "/api/v1/settings/whisper-settings/", "", nil)
	if err != nil {
		return WhisperSettings{}, err
	}
	defer resp.Body.Close()

	var s WhisperSettings
	json.NewDecoder(resp.Body).Decode(&s)
	return s, nil
}

// GetCategories liefert alle verfügbaren Kategorien vom Server.
func (a *App) GetCategories(token string) ([]Category, error) {
	resp, err := a.doRequest("GET", "/api/v1/settings/categories/", token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cats []Category
	json.NewDecoder(resp.Body).Decode(&cats)
	return cats, nil
}

// GetTaskStatus fragt den Status eines Hintergrund-Tasks ab (Transkription + LLM-Summary).
func (a *App) GetTaskStatus(taskID int, token string) (TaskStatus, error) {
	resp, err := a.doRequest("GET", fmt.Sprintf("/api/v1/records/status/%d", taskID), token, nil)
	if err != nil {
		return TaskStatus{}, err
	}
	defer resp.Body.Close()

	var s TaskStatus
	json.NewDecoder(resp.Body).Decode(&s)
	return s, nil
}

// GetRecordBySession liefert das fertige Record (u.a. die LLM-Zusammenfassung) zu einer Session.
func (a *App) GetRecordBySession(sessionID, token string) (RecordDetails, error) {
	resp, err := a.doRequest("GET", fmt.Sprintf("/api/v1/records/by-session/%s", url.QueryEscape(sessionID)), token, nil)
	if err != nil {
		return RecordDetails{}, err
	}
	defer resp.Body.Close()

	var r RecordDetails
	json.NewDecoder(resp.Body).Decode(&r)
	return r, nil
}

// ReprocessWithCategory ändert die Kategorie eines bereits transkribierten Records und
// stößt danach die KI-Zusammenfassung neu an, damit die neue Kategorie-Vorlage greift
// (Kategoriewechsel nach der Aufnahme, analog zum openDox-Web-Frontend). Der
// /summarize-Endpoint ist synchron (liefert den fertigen Text direkt, kein Task/Polling —
// gegen den echten Server verifiziert, task_id bleibt dabei durchgehend null).
func (a *App) ReprocessWithCategory(recordID int, categoryID int, token string) (RecordDetails, error) {
	updateBody := map[string]interface{}{"category_id": categoryID}
	resp, err := a.doRequest("PUT", fmt.Sprintf("/api/v1/records/record/%d", recordID), token, updateBody)
	if err != nil {
		return RecordDetails{}, err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return RecordDetails{}, fmt.Errorf("kategorie konnte nicht aktualisiert werden (status %d)", resp.StatusCode)
	}

	resp, err = a.doRequest("POST", fmt.Sprintf("/api/v1/records/record/%d/summarize", recordID), token, nil)
	if err != nil {
		return RecordDetails{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return RecordDetails{}, fmt.Errorf("neuverarbeitung fehlgeschlagen: %s", string(b))
	}

	var r RecordDetails
	json.NewDecoder(resp.Body).Decode(&r)
	r.ID = recordID
	return r, nil
}

// Quit schließt die Anwendung.
func (a *App) Quit() {
	runtime.Quit(a.ctx)
}

// OpenDashboard öffnet das openDox-Dashboard im Standardbrowser.
func (a *App) OpenDashboard() {
	runtime.BrowserOpenURL(a.ctx, a.opendoxURL+"/dashboard")
}

// CopyToClipboard kopiert Text in die Zwischenablage, ohne einen Tastendruck zu
// simulieren (im Gegensatz zu InsertTranscript) - genutzt für das automatische Kopieren
// nach Abschluss der Verarbeitung, wenn nicht bekannt ist, welches Fenster fokussiert ist.
// Löst zusätzlich eine Desktop-Benachrichtigung aus, da maschinell erstellte Texte vor
// Verwendung geprüft werden müssen (siehe DISCLAIMER.md) und dieser automatische Kopiervorgang
// sonst ganz ohne Nutzerinteraktion abläuft.
func (a *App) CopyToClipboard(text string) error {
	if err := runtime.ClipboardSetText(a.ctx, text); err != nil {
		return err
	}
	go notifyReview()
	return nil
}

// InsertTranscript schreibt den Text in das aktuell fokussierte Fenster.
// viaHotkey unterscheidet die beiden Aufruf-Pfade aus Recorder.tsx: true für
// Cmd+Opt+V und die Diktiermikro-Tasten (nie fokussiert, kein Hide/Show nötig
// - reines Flackern ohne Nutzen), false für den Button-Klick (siehe unten).
//
// Plattform-Unterschied:
//   - macOS: NUR Zwischenablage setzen, kein simuliertes Cmd+V (siehe unten,
//     warum robotgo.KeyTap hier bewusst nicht verwendet wird).
//   - Linux: robotgo.TypeStr() tippt zeichenweise direkt in das Zielfenster.
//     Ein simuliertes Ctrl+V wurde hier zuvor verwendet, kollidierte aber mit
//     dem gerade losgelassenen globalen Hotkey (Ctrl+Alt+V) unter X11 — der
//     Text blieb dann nur in der Zwischenablage.
func (a *App) InsertTranscript(text string, viaHotkey bool) error {
	// Serialisiert gegen gleichzeitige Inserts: Ein Durchlauf (Sleep + zeichenweises
	// Tippen) kann leicht länger als das 700ms-Hotkey-Debounce dauern. Ohne diese
	// Sperre liefen bei mehrfachem Ctrl+Alt+V mehrere robotgo.TypeStr-Aufrufe parallel
	// und interleavten ihre Tastatur-Injektion zu verstümmeltem Text. Ein bereits
	// laufender Insert wird nicht angehängt, sondern schlägt sichtbar fehl (Toast im
	// Frontend) — besser als lautlos zu verwürfeln.
	if !a.insertMu.TryLock() {
		log.Printf("[opendox] InsertTranscript: bereits ein Insert aktiv, ignoriere weiteren Aufruf (viaHotkey=%v)", viaHotkey)
		return fmt.Errorf("Einfügen läuft bereits – bitte kurz warten")
	}
	defer a.insertMu.Unlock()

	// Clipboard wird immer gesetzt — als Fallback (macOS) und als stiller
	// Backup (Linux, falls das Zielfeld getippte Eingaben nicht annimmt).
	runtime.ClipboardSetText(a.ctx, text)
	go notifyReview()

	// Ein Klick auf den "Akzeptieren und einfügen"-Button fokussiert zwangsläufig
	// erst das openDox-Overlay selbst (jeder Mausklick auf ein Fenster holt es in
	// den Fokus), wodurch der Text sonst in openDox statt ins gewünschte Zielprogramm
	// landen würde. Kurzes Verstecken gibt den Fokus an das zuvor aktive Zielfenster
	// zurück. Beim Hotkey-Pfad (und den Diktiermikro-Tasten) war das Overlay ohnehin
	// nie fokussiert, daher wird dort NICHT versteckt - nur ein unnötiges Flackern.
	//
	a.mu.Lock()
	wasVisible := a.visible
	a.mu.Unlock()
	hideForFocus := wasVisible && !viaHotkey
	log.Printf("[opendox] InsertTranscript: start viaHotkey=%v wasVisible=%v hideForFocus=%v len=%d", viaHotkey, wasVisible, hideForFocus, len(text))
	if hideForFocus {
		runtime.WindowHide(a.ctx)
		log.Printf("[opendox] InsertTranscript: WindowHide done")
	}

	// Kurze Pause: gibt dem Fenstermanager Zeit, den Fokuswechsel nach dem
	// WindowHide oben zu vollziehen, und dem Anwender Zeit, den Hotkey-Modifier
	// physisch loszulassen, bevor der Text getippt/eingefügt wird.
	time.Sleep(300 * time.Millisecond)

	var insertErr error
	if goruntime.GOOS == "darwin" {
		// simulatePaste() (paste_darwin.go) statt robotgo.KeyTap("v", "command"): KeyTap löst
		// intern Carbon-TIS/TSM-Tastaturlayout-Auflösung aus (keyCodeForChar), die laut Apple
		// ausschließlich vom Main-Thread aufgerufen werden darf (go-vgo/robotgo#341) — InsertTranscript
		// läuft aber in einem Wails-Dispatcher-Goroutine, nicht auf dem Main-Thread, und das crasht
		// reproduzierbar mit SIGTRAP (go-vgo/robotgo#690, bestätigt auf macOS 26.4/arm64, robotgo
		// v1.0.2, UNABHÄNGIG von der Bedienhilfen-Berechtigung — auch mit nachweislich funktionierender
		// Berechtigung reproduziert). simulatePaste() sendet Cmd+V direkt über CGEventPost mit fest
		// codiertem Tastencode, ohne TIS/TSM, und crasht deshalb nicht.
		simulatePaste()
	} else {
		// Linux/Windows: zeichenweises Tippen. Ein synthetisches Modifier-Release
		// (KeyUp "ctrl"/"alt"/"shift") wurde ausprobiert, hat aber auf Systemen mit
		// nicht-standardmäßigem Keyboard-Mapping den Shift-/CapsLock-Zustand
		// verwürfelt und den Text zerhackt — schlimmer als das Problem, das es lösen
		// sollte. Ein reines Warten ist die robustere Variante.
		robotgo.TypeStr(text)
	}
	log.Printf("[opendox] InsertTranscript: type/paste call returned")

	if hideForFocus {
		runtime.WindowShow(a.ctx)
		a.positionTopRight()
		log.Printf("[opendox] InsertTranscript: WindowShow done")
	}
	return insertErr
}
