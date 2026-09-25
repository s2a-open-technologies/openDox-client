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

//go:build windows

package main

import (
	"os"
	"path/filepath"
)

// exeDirSettingsPath liegt direkt neben der .exe (z.B. C:\Program Files\openDox\
// openDox\settings.json). Der MSI-Installer legt hier eine Vorlage ab, die IT pro
// Kunde anpasst — hat Vorrang vor %AppData% (siehe runtimeSettingsPath), da eine
// Program-Files-Installation den Server üblicherweise zentral vorkonfiguriert.
func exeDirSettingsPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "settings.json"), nil
}
