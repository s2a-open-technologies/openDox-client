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

import (
	"sync"
	"testing"
	"time"
)

type recorder struct {
	mu   sync.Mutex
	seqs []uint32
}

func (r *recorder) clear(seq uint32) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seqs = append(r.seqs, seq)
	return true
}

func (r *recorder) got() []uint32 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]uint32(nil), r.seqs...)
}

func TestScheduleClearsAfterDelay(t *testing.T) {
	r := &recorder{}
	c := NewCleaner(r.clear)
	c.Schedule(7, 20*time.Millisecond)
	time.Sleep(120 * time.Millisecond)
	if got := r.got(); len(got) != 1 || got[0] != 7 {
		t.Errorf("clear calls = %v, want [7]", got)
	}
}

func TestNewScheduleReplacesOldOne(t *testing.T) {
	r := &recorder{}
	c := NewCleaner(r.clear)
	c.Schedule(1, 40*time.Millisecond)
	c.Schedule(2, 40*time.Millisecond)
	time.Sleep(160 * time.Millisecond)
	if got := r.got(); len(got) != 1 || got[0] != 2 {
		t.Errorf("clear calls = %v, want [2] (erster Plan muss verfallen)", got)
	}
}

func TestZeroDelayOrZeroSeqSchedulesNothing(t *testing.T) {
	r := &recorder{}
	c := NewCleaner(r.clear)
	c.Schedule(5, 0)
	c.Schedule(0, 20*time.Millisecond)
	time.Sleep(80 * time.Millisecond)
	c.ClearNow()
	if got := r.got(); len(got) != 0 {
		t.Errorf("clear calls = %v, want none", got)
	}
}

func TestDisablingCancelsPendingClear(t *testing.T) {
	r := &recorder{}
	c := NewCleaner(r.clear)
	c.Schedule(3, 30*time.Millisecond)
	c.Schedule(4, 0) // "nie leeren" ersetzt den alten Plan
	time.Sleep(100 * time.Millisecond)
	c.ClearNow()
	if got := r.got(); len(got) != 0 {
		t.Errorf("clear calls = %v, want none", got)
	}
}

func TestClearNowClearsPendingOnceAndStopsTimer(t *testing.T) {
	r := &recorder{}
	c := NewCleaner(r.clear)
	c.Schedule(9, 40*time.Millisecond)
	c.ClearNow()
	time.Sleep(120 * time.Millisecond)
	if got := r.got(); len(got) != 1 || got[0] != 9 {
		t.Errorf("clear calls = %v, want exactly [9]", got)
	}
	c.ClearNow() // nichts mehr offen
	if got := r.got(); len(got) != 1 {
		t.Errorf("zweites ClearNow darf nicht erneut leeren: %v", got)
	}
}
