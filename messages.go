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
	"strings"
	"sync"
)

// Meldungen, die der Go-Teil selbst erzeugt, müssen zur Oberflächensprache passen.
// Fehler an das Frontend gehen als stabiler Code ("odx:<code>"), den das Frontend über
// seine Übersetzungen (goerr.<code>) anzeigt; nur native Benachrichtigungen des
// Betriebssystems brauchen den Text hier in Go. Die Sprache meldet das Frontend per
// SetUILanguage, sobald sie feststeht oder wechselt.

// codedError ist ein Fehler, dessen Text das Frontend übersetzt. Unbekannte Codes zeigt
// es unverändert an.
type codedError string

func (e codedError) Error() string { return "odx:" + string(e) }

const (
	errInsertBusy      = codedError("insert_busy")
	errClipboardNA     = codedError("clipboard_unavailable")
	errModifiersHeld   = codedError("modifiers_held")
	errSendInputFailed = codedError("sendinput_failed")
	oidcErrNoToken     = "odx:oidc_no_token"
	oidcErrTimeout     = "odx:oidc_timeout"
	defaultUILanguage  = "de"
	msgNotifyReviewKey = "notify_review"
)

var goMessages = map[string]map[string]string{
	"de": {msgNotifyReviewKey: "Text eingefügt – bitte vor Verwendung prüfen."},
	"en": {msgNotifyReviewKey: "Text inserted – please review before use."},
	"es": {msgNotifyReviewKey: "Texto insertado: revíselo antes de usarlo."},
}

var (
	uiLangMu sync.RWMutex
	uiLang   string
)

// SetUILanguage merkt sich die vom Frontend gewählte Oberflächensprache (de/en/es).
func (a *App) SetUILanguage(lang string) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if len(lang) > 2 {
		lang = lang[:2]
	}
	if _, ok := goMessages[lang]; !ok {
		return
	}
	uiLangMu.Lock()
	uiLang = lang
	uiLangMu.Unlock()
}

// goMessage liefert den Text in der aktuellen Oberflächensprache (Fallback Deutsch).
func goMessage(key string) string {
	uiLangMu.RLock()
	lang := uiLang
	uiLangMu.RUnlock()
	if m, ok := goMessages[lang]; ok {
		return m[key]
	}
	return goMessages[defaultUILanguage][key]
}
