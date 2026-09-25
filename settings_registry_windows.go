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

import "golang.org/x/sys/windows/registry"

// registryServerURL liest die vom MSI-Installer deklarativ gesetzte Server-URL aus
// HKLM\Software\openDox\openDox\ServerURL (siehe ServerUrlRegistry-Component in
// build/windows/installer/opendox-client.wxs — RegistryValue's Value="[SERVERURL]" wird
// von Windows Installer selbst beim Schreiben formatiert, ohne CustomAction). Existiert
// der Schlüssel nicht (z.B. lokale Entwicklung oder eine Nicht-MSI-Installation), liefert
// dies ok=false, sodass loadSettings() auf die bestehenden Fallbacks zurückfällt.
func registryServerURL() (string, bool) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `Software\openDox\openDox`, registry.QUERY_VALUE)
	if err != nil {
		return "", false
	}
	defer k.Close()

	v, _, err := k.GetStringValue("ServerURL")
	if err != nil || v == "" {
		return "", false
	}
	return v, true
}
