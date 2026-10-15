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
	"time"
)

// Cleaner leert die Zwischenablage zeitverzögert, aber nur, solange sie noch den vom
// Client gesetzten Inhalt enthält. Wie das "Leeren, falls unverändert" konkret
// geschieht, bestimmt die übergebene Funktion (unter Windows: Vergleich der
// Zwischenablage-Sequenznummer), damit die Zeitsteuerung hier ohne Betriebssystem
// testbar bleibt.
type Cleaner struct {
	mu    sync.Mutex
	timer *time.Timer
	seq   uint32
	clear func(seq uint32) bool
}

// NewCleaner erzeugt einen Cleaner; clearIfUnchanged leert die Zwischenablage, wenn
// deren Sequenznummer noch seq ist, und meldet, ob geleert wurde.
func NewCleaner(clearIfUnchanged func(seq uint32) bool) *Cleaner {
	return &Cleaner{clear: clearIfUnchanged}
}

// Schedule plant das Leeren für seq nach dem Zeitraum after und ersetzt einen noch
// laufenden früheren Plan. seq == 0 (Plattform ohne Sequenznummer) oder after <= 0
// ("nie leeren") plant nichts und verwirft den alten Plan.
func (c *Cleaner) Schedule(seq uint32, after time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopLocked()
	if seq == 0 || after <= 0 {
		return
	}
	c.seq = seq
	c.timer = time.AfterFunc(after, func() { c.fire(seq) })
}

func (c *Cleaner) fire(seq uint32) {
	c.mu.Lock()
	if c.seq != seq {
		// Inzwischen durch einen neueren Plan ersetzt oder bereits geleert.
		c.mu.Unlock()
		return
	}
	c.seq = 0
	c.timer = nil
	c.mu.Unlock()
	c.clear(seq)
}

// ClearNow leert sofort, falls noch ein Plan offen ist (z. B. beim Beenden des Clients).
func (c *Cleaner) ClearNow() {
	c.mu.Lock()
	seq := c.seq
	c.stopLocked()
	c.mu.Unlock()
	if seq != 0 {
		c.clear(seq)
	}
}

func (c *Cleaner) stopLocked() {
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
	c.seq = 0
}
