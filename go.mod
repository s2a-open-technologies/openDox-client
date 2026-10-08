module opendox-client

go 1.26.0

require (
	git.sr.ht/~jackmordaunt/go-toast/v2 v2.0.3
	github.com/go-vgo/robotgo v1.3.2
	github.com/ricknicius99/hid v0.0.0-20231128165240-e77f6da9e619
	github.com/wailsapp/wails/v2 v2.16.0
	golang.design/x/hotkey v0.6.1
	golang.org/x/sys v0.48.0
)

require (
	github.com/bep/debounce v1.2.1 // indirect
	github.com/dblohm7/wingoes v0.0.0-20260526185140-fb298caac7ca // indirect
	github.com/ebitengine/purego v0.11.1 // indirect
	github.com/gen2brain/shm v0.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/jchv/go-winloader v0.0.0-20210711035445-715c2860da7e // indirect
	github.com/jezek/xgb v1.3.1 // indirect
	github.com/jezek/xgbutil v0.0.0-20260124183602-9fd151d6a51a // indirect
	github.com/labstack/echo/v4 v4.13.3 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/leaanthony/go-ansi-parser v1.6.1 // indirect
	github.com/leaanthony/gosod v1.0.4 // indirect
	github.com/leaanthony/slicer v1.6.0 // indirect
	github.com/leaanthony/u v1.1.1 // indirect
	github.com/lufia/plan9stats v0.0.0-20260802145828-341c2f0c90b5 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/otiai10/gosseract/v2 v2.4.1 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/power-devops/perfstat v0.0.0-20260916203055-22a1a467d9f0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/samber/lo v1.49.1 // indirect
	github.com/shirou/gopsutil/v4 v4.26.9 // indirect
	github.com/tailscale/win v0.0.0-20260619195133-2d76c33a64c1 // indirect
	github.com/tklauser/go-sysconf v0.4.0 // indirect
	github.com/tklauser/numcpus v0.12.0 // indirect
	github.com/tkrajina/go-reflector v0.5.8 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/vcaesar/go-wayland v0.40.0 // indirect
	github.com/vcaesar/gops v0.43.0 // indirect
	github.com/vcaesar/imgo v0.44.0 // indirect
	github.com/vcaesar/keycode v0.20.0 // indirect
	github.com/vcaesar/screenshot v0.40.0 // indirect
	github.com/vcaesar/tt v0.40.0 // indirect
	github.com/wailsapp/go-webview2 v1.0.22 // indirect
	github.com/wailsapp/mimetype v1.4.1 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/exp v0.0.0-20260908205506-85c1c2202aba // indirect
	golang.org/x/image v0.46.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

// replace github.com/wailsapp/wails/v2 v2.13.0 => /home/crizz/go/pkg/mod

// Local copy of github.com/ricknicius99/hid (karalabe/hid fork) patched to fix a
// macOS-only crash: hid_enumerate() calls IORegistryEntryGetPath() on the io_service_t
// returned by IOHIDDeviceGetService() without checking for IO_OBJECT_NULL first. On
// older macOS an invalid port there just failed softly (KERN_INVALID_ARGUMENT); on
// macOS 26 (Tahoe)'s hardened mach-port guards, the same call is now a fatal EXC_GUARD/
// GUARD_TYPE_MACH_PORT crash ("INVALID_OPTIONS on mach port 0"). See
// third_party/hid/hidapi/mac/hid.c for the fix. Upstream karalabe/hid is archived and
// unmaintained; this fork doesn't fix the bug either, hence vendoring the patch locally
// instead of waiting on upstream.
replace github.com/ricknicius99/hid => ./third_party/hid

// Local copy of golang.design/x/hotkey patched to enable X11 Detectable
// Autorepeat (XkbSetDetectableAutoRepeat) on every display connection the
// library opens. Without it, holding a hotkey floods KeyPress events, which
// the app-side 700ms debounce can only mask, not prevent. Upstream is
// unmaintained (last release 2023), hence vendoring the patch locally.
replace golang.design/x/hotkey => ./third_party/hotkey
