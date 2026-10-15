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
	"os"
	"path/filepath"
	"strings"
)

// exeDirSettingsPath liefert die settings.json im selben Ordner wie das Programm, z.B.
// C:\Program Files\openDox\settings.json (der Installer legt dort eine Vorlage ab, die IT pro
// Kunde anpasst) oder build/bin/settings.json unter Linux. Hat Vorrang vor dem
// Benutzer-Konfigordner (siehe runtimeSettingsPath).
func exeDirSettingsPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(settingsDirForExe(exe), "settings.json"), nil
}

// settingsDirForExe ist der Ordner, in dem neben dem Programm die settings.json gesucht wird.
// Bei einem macOS-Bundle (…/openDox.app/Contents/MacOS/opendox-client) ist das der Ordner, in
// dem die .app liegt: eine Datei im Bundle würde dessen codesign-Signatur brechen (siehe
// build-macos.sh zur TCC-/Codesign-Fragilität).
func settingsDirForExe(exe string) string {
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "MacOS" && filepath.Base(filepath.Dir(dir)) == "Contents" &&
		strings.HasSuffix(filepath.Dir(filepath.Dir(dir)), ".app") {
		return filepath.Dir(filepath.Dir(filepath.Dir(dir)))
	}
	return dir
}
