#!/usr/bin/env bash
# Builds production binaries for Linux (native) and Windows (cross-compiled via
# mingw-w64), including the NSIS installer for Windows. Does NOT launch the app.
#
# macOS is intentionally not built here: Wails v2 has no official cross-compile
# support for macOS targets from a non-Mac host. Build it natively on a Mac with:
#   wails build -platform darwin/arm64
set -eu
cd "$(dirname "$0")"

# Über SSH ist DISPLAY nicht gesetzt; manche cgo-Schritte (X11-gebundene Deps wie
# hotkey/webkit) brauchen die Variable beim Bauen. Fallback auf das lokale Display.
export DISPLAY="${DISPLAY:-:0}"

echo "==> Building Linux binary (wails build -tags webkit2_41 -platform linux/amd64)"
wails build -clean -tags webkit2_41 -platform linux/amd64
echo "    -> build/bin/opendox-client"
 
if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1 && command -v makensis >/dev/null 2>&1; then
    echo "==> Building Windows binary + NSIS installer (wails build -platform windows/amd64 -nsis)"
    # No -clean here: it wipes the whole build/bin dir, which would delete the
    # Linux binary built above. CGO_ENABLED must be explicit — Go disables cgo
    # by default on cross-builds, which breaks robotgo (used for Ctrl+Alt+A typing).
    CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 wails build -platform windows/amd64 -nsis
    echo "    -> build/bin/opendox-client.exe + NSIS installer under build/bin/"
else
    echo "==> Skipping Windows build: x86_64-w64-mingw32-gcc and/or makensis not found."
    echo "    Install with: sudo apt install gcc-mingw-w64-x86-64 nsis"
    exit 1
fi

echo "==> Skipping macOS build: not supported cross-compiling from Linux."
echo "    Build natively on a Mac with: wails build -platform darwin/arm64"

echo "==> Done."
