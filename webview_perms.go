//go:build linux

package main

/*
#cgo pkg-config: webkit2gtk-4.1 gtk+-3.0

#include <gtk/gtk.h>
#include <webkit2/webkit2.h>

// Auto-allow every permission request emitted by the WebView
// (microphone, camera, geolocation, notifications, ...).
static gboolean permission_requested_cb(WebKitWebView *webview,
                                        WebKitPermissionRequest *request,
                                        gpointer data) {
    webkit_permission_request_allow(request);
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
