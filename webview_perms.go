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

//go:build linux

package main

/*
#cgo pkg-config: webkit2gtk-4.1 gtk+-3.0

#include <gtk/gtk.h>
#include <webkit2/webkit2.h>

// Auto-allow only microphone/camera access requests (getUserMedia) emitted by
// the WebView — the app only ever needs microphone access for dictation.
// Every other permission type (geolocation, notifications, ...) is denied.
static gboolean permission_requested_cb(WebKitWebView *webview,
                                        WebKitPermissionRequest *request,
                                        gpointer data) {
    if (WEBKIT_IS_USER_MEDIA_PERMISSION_REQUEST(request)) {
        webkit_permission_request_allow(request);
    } else {
        webkit_permission_request_deny(request);
    }
    return TRUE;
}

// Recursively walk the GTK widget tree and connect the permission-request
// signal to the first WebKitWebView we find. Safe to call multiple times —
// the "perms-connected" data key guards against double-connecting.
static void find_and_connect_cb(GtkWidget *widget, gpointer data) {
    if (WEBKIT_IS_WEB_VIEW(widget)) {
        if (g_object_get_data(G_OBJECT(widget), "perms-connected") == NULL) {
            g_object_set_data(G_OBJECT(widget), "perms-connected", GINT_TO_POINTER(1));
            g_signal_connect(G_OBJECT(widget), "permission-request",
                             G_CALLBACK(permission_requested_cb), NULL);

            // Also explicitly enable media-stream in the settings (default is
            // usually TRUE, but we enforce it to be safe).
            WebKitSettings *settings = webkit_web_view_get_settings(WEBKIT_WEB_VIEW(widget));
            webkit_settings_set_enable_media_stream(settings, TRUE);
        }
        return;
    }
    if (GTK_IS_CONTAINER(widget)) {
        gtk_container_foreach(GTK_CONTAINER(widget), find_and_connect_cb, NULL);
    }
}

// Walk every toplevel window's widget tree and hook the permission handler.
void setup_webview_permissions_c(void) {
    GList *toplevels = gtk_window_list_toplevels();
    for (GList *l = toplevels; l != NULL; l = l->next) {
        find_and_connect_cb(GTK_WIDGET(l->data), NULL);
    }
    g_list_free(toplevels);
}
*/
import "C"

// setupWebviewPermissions connects an auto-approve handler to the WebView's
// permission-request signal so that getUserMedia (microphone) is granted
// without a dialog. Must be called on the GTK main thread.
func setupWebviewPermissions() {
	C.setup_webview_permissions_c()
}
