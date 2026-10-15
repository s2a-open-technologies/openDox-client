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
#cgo LDFLAGS: -framework WebKit -framework Foundation
#import <WebKit/WebKit.h>
#import <objc/runtime.h>

// Wails' eigene WailsContext-Klasse (internal/frontend/desktop/darwin/WailsContext.m,
// wails v2.15.0) setzt sich selbst als WKUIDelegate ([self.webview.UIDelegate = self]),
// implementiert darin aber NICHT die optionale Methode
// webView:requestMediaCapturePermissionForOrigin:initiatedByFrame:type:decisionHandler:.
// Ohne diese Methode zeigt WKWebView bei jedem getUserMedia()-Aufruf sein EIGENES,
// separates Berechtigungs-Popup — zusätzlich zur (bereits erteilten) macOS-System-
// Mikrofonberechtigung, und OHNE diese über App-Neustarts hinweg zu merken (dokumentiertes
// WebKit-Verhalten, kein Wails-Bug: https://bugs.webkit.org/show_bug.cgi?id=220416).
//
// Da wir Wails' Objective-C-Quelle nicht patchen wollen/können, wird die fehlende Methode
// hier zur Laufzeit per Objective-C-Runtime auf die WailsContext-KLASSE addiert (nicht
// überschrieben — falls eine künftige Wails-Version die Methode selbst mitbringt, wird das
// hier übersprungen). Analog zu webview_perms.go, das auf Linux dieselbe Lücke über ein
// GTK-Signal schließt.
void setup_webview_permissions_darwin(void) {
    Class cls = objc_getClass("WailsContext");
    if (cls == nil) {
        return;
    }

    SEL selector = @selector(webView:requestMediaCapturePermissionForOrigin:initiatedByFrame:type:decisionHandler:);
    if (class_getInstanceMethod(cls, selector) != NULL) {
        return; // Methode existiert schon (z.B. neuere Wails-Version) — nichts tun.
    }

    // imp_implementationWithBlock erzeugt aus einem Objective-C-Block eine IMP mit
    // korrekter Aufrufkonvention (inkl. des Block-Parameters decisionHandler) — deutlich
    // robuster als eine reine C-Funktion mit von Hand nachgebauter Block-Struktur.
    id block = ^(id self, WKWebView *webView, WKSecurityOrigin *origin, WKFrameInfo *frame,
                 NSInteger type, void (^decisionHandler)(NSInteger decision)) {
        decisionHandler(1); // WKPermissionDecisionGrant
    };
    IMP imp = imp_implementationWithBlock(block);

    char typeEncoding[32];
    snprintf(typeEncoding, sizeof(typeEncoding), "v@:@@@%s@?", @encode(NSInteger));
    class_addMethod(cls, selector, imp, typeEncoding);
}
*/
import "C"

// setupWebviewPermissions klinkt die fehlende WKUIDelegate-Methode für
// Medienzugriffs-Anfragen (Mikrofon/Kamera) in Wails' Fensterklasse ein, damit
// getUserMedia() nicht bei jedem Aufruf erneut WebKits eigenes Berechtigungs-
// Popup zeigt. Muss vor dem ersten getUserMedia()-Aufruf laufen; wird analog
// zu Linux in domReady() aufgerufen.
func setupWebviewPermissions() {
	C.setup_webview_permissions_darwin()
}
