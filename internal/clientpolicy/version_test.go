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

package clientpolicy

import "testing"

func TestNewerVersion(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"0.9.0", "0.8.5", true},
		{"0.8.10", "0.8.9", true}, // numerisch, nicht lexikalisch
		{"1.0.0", "0.99.99", true},
		{"v0.9.0", "0.8.5", true},
		{"0.8.5", "0.8.5", false},
		{"0.8.4", "0.8.5", false},
		{"", "0.8.5", false},
		{"0.9.0", "", false},
		{"0.9", "0.8.5", false},
		{"0.9.0-rc1", "0.8.5", false},
		{"kaputt", "0.8.5", false},
	}
	for _, c := range cases {
		if got := NewerVersion(c.latest, c.current); got != c.want {
			t.Errorf("NewerVersion(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestMergeKeepsServerLatestVersion(t *testing.T) {
	got := Merge(Settings{LatestClientVersion: "0.9.0"}, Settings{LatestClientVersion: "9.9.9"})
	if got.LatestClientVersion != "0.9.0" {
		t.Errorf("LatestClientVersion = %q, want Server-Wert 0.9.0", got.LatestClientVersion)
	}
}
