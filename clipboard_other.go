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

//go:build !windows

package main

import (
	"context"
	"log"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// setClipboardText setzt außerhalb von Windows weiter Klartext über Wails (bisheriges
// Verhalten). Es gibt dort keine Markierung "nicht in den Verlauf" und keine
// Sequenznummer — die Rückgabe 0 sorgt dafür, dass nichts automatisch geleert wird.
// Fehler werden wie bisher nicht an den Aufrufer gemeldet.
func setClipboardText(ctx context.Context, text string, _ bool) (uint32, error) {
	if err := runtime.ClipboardSetText(ctx, text); err != nil {
		log.Printf("[opendox] Zwischenablage setzen fehlgeschlagen: %v", err)
	}
	return 0, nil
}

func clearClipboardIfUnchanged(uint32) bool { return false }

// usesClipboardPath: macOS (Cmd+V) und Linux (Strg+V) fügen über die Zwischenablage ein.
func usesClipboardPath() bool { return true }
