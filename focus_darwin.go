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

//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#include <stdatomic.h>

// PID der zuletzt aktivierten fremden App (0 = noch keine bekannt). Wird vom
// NSWorkspace-Observer auf dem Main-Thread geschrieben und aus InsertTranscript
// (Wails-Goroutine) gelesen, daher atomar.
static _Atomic int lastForeignPID = 0;
static id activationObserver = nil;

static void rememberApp(NSRunningApplication *app) {
    if (app == nil) {
        return;
    }
    pid_t pid = app.processIdentifier;
    if (pid > 0 && pid != getpid()) {
        atomic_store(&lastForeignPID, (int)pid);
    }
}

// installActivationObserver merkt sich ab jetzt jede aktivierte fremde App.
// Idempotent; läuft auf dem Main-Thread, da NSWorkspace/NSNotificationCenter-
// Registrierung und die Startwert-Abfrage dort hingehören.
void installActivationObserver() {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (activationObserver != nil) {
            return;
        }
        NSWorkspace *ws = [NSWorkspace sharedWorkspace];
        rememberApp(ws.frontmostApplication);
        activationObserver = [ws.notificationCenter
            addObserverForName:NSWorkspaceDidActivateApplicationNotification
                        object:nil
                         queue:[NSOperationQueue mainQueue]
                    usingBlock:^(NSNotification *note) {
                        rememberApp(note.userInfo[NSWorkspaceApplicationKey]);
                    }];
    });
}

// activateLastForeignApp holt die zuletzt aktive fremde App wieder nach vorn.
// Liefert die PID oder 0, wenn keine (mehr) bekannt ist. Ab macOS 14 ist
// Aktivierung kooperativ: wir (gerade aktiv, weil der Button geklickt wurde)
// müssen die Aktivierung explizit abgeben, sonst ignoriert das System sie.
int activateLastForeignApp() {
    int pid = atomic_load(&lastForeignPID);
    if (pid <= 0) {
        return 0;
    }
    __block int result = 0;
    dispatch_sync(dispatch_get_main_queue(), ^{
        NSRunningApplication *app =
            [NSRunningApplication runningApplicationWithProcessIdentifier:(pid_t)pid];
        if (app == nil || app.terminated) {
            return;
        }
        if (@available(macOS 14.0, *)) {
            [NSApp yieldActivationToApplication:app];
            [app activateWithOptions:0];
        } else {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
            [app activateWithOptions:NSApplicationActivateIgnoringOtherApps];
#pragma clang diagnostic pop
        }
        result = pid;
    });
    return result;
}
*/
import "C"

import "log"

// setOverlayNoActivate: macOS kennt kein WS_EX_NOACTIVATE für normale Fenster.
// Stattdessen merkt sich ein NSWorkspace-Observer die zuletzt aktive fremde App,
// und restoreInsertFocus() aktiviert sie vor dem Cmd+V gezielt wieder. Liefert
// true, damit InsertTranscript das Overlay NICHT versteckt: orderOut gibt den Fokus
// auf macOS nicht an die vorherige App zurück (openDox bliebe aktive App ohne
// Fenster), Cmd+V ginge ins Leere und das Overlay blieb unsichtbar.
func setOverlayNoActivate(enabled bool) bool {
	if enabled {
		C.installActivationObserver()
	}
	return enabled
}

// restoreInsertFocus gibt den Fokus vor dem Button-Insert an das vorherige
// Programm zurück. Nur die PID wird geloggt — nie Fenstertitel (Patientendaten).
func restoreInsertFocus() {
	if pid := C.activateLastForeignApp(); pid == 0 {
		log.Printf("[opendox] restoreInsertFocus: kein vorheriges Programm bekannt")
	} else {
		log.Printf("[opendox] restoreInsertFocus: pid %d aktiviert", int(pid))
	}
}
