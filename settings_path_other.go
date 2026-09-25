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

//go:build !windows

package main

import "os"

// exeDirSettingsPath ist auf macOS/Linux ein No-Op-Stub — dort bleibt es bei
// %AppData%-Äquivalent via os.UserConfigDir() (siehe runtimeSettingsPath). Kein
// exe-Verzeichnis-Lookup auf macOS: eine zusätzliche Datei neben der .exe innerhalb
// von opendox-client.app/Contents/MacOS/ würde die codesign --deep-Signatur des
// Bundles gefährden (siehe build-macos.sh zur TCC-/Codesign-Fragilität). Linux hat
// kein Program-Files-Äquivalent, das diesen Lookup bräuchte.
func exeDirSettingsPath() (string, error) {
	return "", os.ErrNotExist
}
