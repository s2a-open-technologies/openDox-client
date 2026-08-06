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
	"os/exec"
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

// --- App Struct ---

type App struct {
	ctx        context.Context
	opendoxURL string

	mu      sync.Mutex
	visible bool
}

func NewApp() *App {
	return &App{opendoxURL: "http://localhost:20000", visible: true}
}

// --- Lifecycle ---

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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
	// hotkeyMods ist plattformspezifisch definiert (hotkey_mod_*.go) — sowohl die
	// Modifier-Konstanten als auch die übliche Tastenkombination (Ctrl+Alt vs.
	// Cmd+Option auf macOS) unterscheiden sich pro OS.
	mods := hotkeyMods
	a.registerHotkey(mods, hotkey.KeyR, eventToggleRecord)
	a.registerHotkey(mods, hotkey.KeyP, eventTogglePause)
	a.registerHotkey(mods, hotkey.KeyV, eventAcceptInsert)
	a.registerWindowHotkey(mods, hotkey.KeyH) // Ctrl+Alt+H: Overlay ein-/ausblenden
}

// registerHotkey registriert eine einzelne Tastenkombination und emittiert bei jedem
// (entprellten) Tastendruck das angegebene Wails-Event.
func (a *App) registerHotkey(mods []hotkey.Modifier, key hotkey.Key, eventName string) {
	hk := hotkey.New(mods, key)
	if err := hk.Register(); err != nil {
		log.Printf("[opendox] failed to register global hotkey for %s: %v", eventName, err)
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
	return fmt.Sprintf("ws://localhost:20000/api/v1/records/ws/record?session_id=%s&user_id=%s&token=%s",
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

// notifyReview zeigt eine native Desktop-Benachrichtigung als Erinnerung, den kopierten
// Text vor Verwendung zu prüfen. Best-effort: falls notify-send fehlt oder fehlschlägt,
// wird das nur geloggt, da die Zwischenablage bereits erfolgreich gesetzt wurde.
func notifyReview() {
	cmd := exec.Command("notify-send", "openDox", "Text eingefügt – bitte vor Verwendung prüfen.")
	if err := cmd.Run(); err != nil {
		log.Printf("[opendox] notify-send failed: %v", err)
	}
}

// InsertTranscript schreibt den Text direkt über die Tastatur in das aktuell
// fokussierte Fenster (robotgo.TypeStr), sodass ein separater Einfügen-Schritt
// entfällt. Die Zwischenablage wird zusätzlich als stiller Fallback gesetzt,
// falls das Zielfeld getippte Eingaben nicht annimmt.
//
// Hintergrund: vorher wurde Ctrl+V simuliert, was beim globalen Hotkey
// Ctrl+Alt+V unter X11 unzuverlässig war (Modifier-Konflikt mit dem gerade
// losgelassenen Hotkey) - der Text blieb dann nur in der Zwischenablage.
func (a *App) InsertTranscript(text string) error {
	runtime.ClipboardSetText(a.ctx, text)
	go notifyReview()
	// Kurze Pause statt eines synthetischen Modifier-Release: robotgo.KeyUp("ctrl"/"alt"/"shift")
	// wurde hier kurz ausprobiert, hat aber auf Systemen mit nicht-standardmäßigem Keyboard-Mapping
	// (siehe hotkeyMods-Kommentar in hotkey_mod_linux.go) den Shift-/CapsLock-Zustand verwürfelt und
	// dadurch den getippten Text zerhackt - schlimmer als das Problem, das es lösen sollte. Ein
	// reines Warten (kein zusätzliches synthetisches Tastenereignis) gibt dem Anwender Zeit, Ctrl+Alt
	// vom gerade gedrückten Hotkey tatsächlich physisch loszulassen, bevor robotgo.TypeStr tippt -
	// ohne das Tastatur-Modifier-Bit selbst anzufassen.
	time.Sleep(300 * time.Millisecond)
	robotgo.TypeStr(text)
	return nil
}
