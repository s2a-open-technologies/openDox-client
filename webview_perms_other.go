//go:build !linux

package main

// setupWebviewPermissions is a no-op on non-Linux platforms — the
// WebKitGTK permission-prompt problem this works around only exists on
// Linux; Windows (WebView2) and macOS (WKWebView) handle getUserMedia
// permissions differently.
func setupWebviewPermissions() {}
