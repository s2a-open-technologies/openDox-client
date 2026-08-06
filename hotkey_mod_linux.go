//go:build linux

package main

import "golang.design/x/hotkey"

// hotkeyMods = Ctrl+Alt auf diesem System (per "xmodmap -pm" verifiziert; X11 nennt
// Alt "Mod1", die Zuordnung ist nicht plattformweit garantiert, siehe
// golang.design/x/hotkey Paket-Doku).
var hotkeyMods = []hotkey.Modifier{hotkey.ModCtrl, hotkey.Mod1}
