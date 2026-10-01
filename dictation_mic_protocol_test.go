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

import "testing"

func TestMatchDictationMicProfile(t *testing.T) {
	cases := []struct {
		name                string
		vendorID, productID uint16
		wantProfile         string // "" means no match expected
	}{
		{"SpeechMike wired LFH/SMP", 0x0911, 0x0c1c, "SpeechMike/PowerMic IV"},
		{"SpeechMike Premium Air", 0x0911, 0x0c1d, "SpeechMike/PowerMic IV"},
		{"SpeechOne/Ambient", 0x0911, 0x0c1e, "SpeechMike/PowerMic IV"},
		{"SpeechMike Browser/Gamepad mode HID interface", 0x0911, 0x0fa0, "SpeechMike/PowerMic IV"},
		{"PowerMic IV", 0x0554, 0x0064, "SpeechMike/PowerMic IV"},
		{"PowerMic III", 0x0554, 0x1001, "PowerMic III"},
		{"unrelated vendor", 0x1234, 0x5678, ""},
		{"Philips vendor, unknown product", 0x0911, 0x9999, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			profile := matchDictationMicProfile(c.vendorID, c.productID)
			if c.wantProfile == "" {
				if profile != nil {
					t.Fatalf("expected no match, got profile %q", profile.name)
				}
				return
			}
			if profile == nil {
				t.Fatalf("expected profile %q, got no match", c.wantProfile)
			}
			if profile.name != c.wantProfile {
				t.Fatalf("expected profile %q, got %q", c.wantProfile, profile.name)
			}
		})
	}
}

func TestParseSpeechMikeReport(t *testing.T) {
	cases := []struct {
		name     string
		report   []byte
		wantOK   bool
		wantMask micButtonMask
	}{
		// Command byte 0x80 (BUTTON_PRESS_EVENT), bitmask = LE uint16 at offset 7.
		{"record pressed (bit8 -> 0x0100)", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x01}, true, micRecord},
		{"stop pressed (bit9 -> 0x0200)", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x02}, true, micStop},
		{"rewind pressed (bit12 -> 0x1000)", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x10}, true, micRewind},
		{"ins/ovr pressed (bit14 -> 0x4000)", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x40}, true, micInsOvr},
		{"record+stop both set", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x03}, true, micRecord | micStop},
		{"no buttons pressed", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x00, 0x00}, true, 0},
		{"unrelated bit ignored (F1_A, bit1)", []byte{0x80, 0, 0, 0, 0, 0, 0, 0x02, 0x00}, true, 0},
		{"wrong command byte (motion event 0x9e) ignored", []byte{0x9e, 0, 0, 0, 0, 0, 0, 0, 0}, false, 0},
		{"report too short", []byte{0x80, 0, 0}, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mask, ok := parseSpeechMikeReport(c.report)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if ok && mask != c.wantMask {
				t.Fatalf("mask = %v, want %v", mask, c.wantMask)
			}
		})
	}
}

func TestParsePowerMic3Report(t *testing.T) {
	cases := []struct {
		name     string
		report   []byte
		wantOK   bool
		wantMask micButtonMask
	}{
		// No command byte check, bitmask = LE uint16 at offset 1.
		{"record pressed (bit2 -> 0x0004)", []byte{0, 0x04, 0x00}, true, micRecord},
		{"rewind pressed (bit4 -> 0x0010)", []byte{0, 0x10, 0x00}, true, micRewind},
		{"no buttons pressed", []byte{0, 0x00, 0x00}, true, 0},
		{"unrelated bit ignored (TRANSCRIBE, bit0)", []byte{0, 0x01, 0x00}, true, 0},
		{"report too short", []byte{0, 0x04}, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mask, ok := parsePowerMic3Report(c.report)
			if ok != c.wantOK {
				t.Fatalf("ok = %v, want %v", ok, c.wantOK)
			}
			if ok && mask != c.wantMask {
				t.Fatalf("mask = %v, want %v", mask, c.wantMask)
			}
		})
	}
}

func TestRisingEdge(t *testing.T) {
	cases := []struct {
		name             string
		prev, cur, want micButtonMask
	}{
		{"press from idle", 0, micRecord, micRecord},
		{"held (no new edge)", micRecord, micRecord, 0},
		{"release (no edge)", micRecord, 0, 0},
		{"second button added while first held", micRecord, micRecord | micStop, micStop},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := risingEdge(c.prev, c.cur); got != c.want {
				t.Fatalf("risingEdge(%v, %v) = %v, want %v", c.prev, c.cur, got, c.want)
			}
		})
	}
}

func TestFallingEdge(t *testing.T) {
	cases := []struct {
		name             string
		prev, cur, want micButtonMask
	}{
		{"press (no edge)", 0, micRecord, 0},
		{"held (no edge)", micRecord, micRecord, 0},
		{"release", micRecord, 0, micRecord},
		{"one of two released", micRecord | micStop, micStop, micRecord},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := fallingEdge(c.prev, c.cur); got != c.want {
				t.Fatalf("fallingEdge(%v, %v) = %v, want %v", c.prev, c.cur, got, c.want)
			}
		})
	}
}
