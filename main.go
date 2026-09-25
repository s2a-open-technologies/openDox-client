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
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Subprozess-Modus für den isolierten Diktiermikro-Scan — siehe
	// dictation_mic_scan_helper.go für den Grund (crashfeste Isolation eines instabilen
	// macOS-IOKit-Aufrufs). Muss vor jeder Wails-/GUI-Initialisierung geprüft werden.
	if os.Getenv("ODX_DICTATION_MIC_SCAN_HELPER") == "1" {
		runDictationMicScanHelper()
		return
	}

	// Konsolen-Ausgabe stillstellen (außer bei --debug / Dev-Builds), bevor
	// GTK/WebKitGTK/JSC als cgo-Libs laden und ihre Meldungen nach fd 1/2
	// schreiben. Muss nach dem Scan-Helper-Check stehen — der Helper liefert
	// sein JSON über stdout und darf niemals umgeleitet werden.
	if !debugRequested() {
		preSilence()
		silenceNativeConsole()
	}
	setupConsole()

	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "openDox",
		Width:  360,
		Height: 520,
		AlwaysOnTop: true,
		Frameless: false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour:         &options.RGBA{R: 24, G: 24, B: 27, A: 255},
		EnableDefaultContextMenu: true,
		OnStartup:                app.startup,
		OnDomReady:               app.domReady,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
