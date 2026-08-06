//go:build darwin

package main

import "golang.design/x/hotkey"

// hotkeyMods = Cmd+Option — auf macOS ist Ctrl+Alt für globale App-Shortcuts
// unüblich, ⌘+⌥ ist die dortige Konvention.
var hotkeyMods = []hotkey.Modifier{hotkey.ModCmd, hotkey.ModOption}
