#!/usr/bin/env bash
# build-windows-msi.sh — Windows-Cross-Build + MSI-Installer via wixl
#
# Baut das Windows-Binary mit demselben mingw-w64-Cross-Compile wie compile.sh
# (siehe dort für Details zu CGO_ENABLED/robotgo) und packt es anschließend mit
# `wixl` (Paket msitools, natives Linux-Tool — kein Wine nötig) zu einem .msi statt
# des von Wails mitgelieferten NSIS-Installers. Additiv zu compile.sh's -nsis-Pfad,
# kein Ersatz dafür — das MSI ist für GPO-/SCCM-/Intune-artige
# Enterprise-Verteilung gedacht, wo .msi das native Format ist.
#
# Voraussetzung (falls wixl fehlt):
#   sudo apt install msitools
export DISPLAY="${DISPLAY:-:0}"
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
    echo "Fehler: x86_64-w64-mingw32-gcc nicht gefunden." >&2
    echo "  Installieren mit: sudo apt install gcc-mingw-w64-x86-64" >&2
    exit 1
fi

if ! command -v wixl >/dev/null 2>&1; then
    echo "Fehler: wixl nicht gefunden." >&2
    echo "  Installieren mit: sudo apt install msitools" >&2
    exit 1
fi

INSTALLED_WAILS_VERSION=$(wails version 2>/dev/null | head -n1 || true)
REQUIRED_WAILS_VERSION=$(go list -m -f '{{.Version}}' github.com/wailsapp/wails/v2 2>/dev/null || true)
if [[ -n "$INSTALLED_WAILS_VERSION" && -n "$REQUIRED_WAILS_VERSION" && "$INSTALLED_WAILS_VERSION" != "$REQUIRED_WAILS_VERSION" ]]; then
    echo "Warnung: installierte wails-CLI ($INSTALLED_WAILS_VERSION) weicht von go.mod ($REQUIRED_WAILS_VERSION) ab."
    echo "  Angleichen mit: go install github.com/wailsapp/wails/v2/cmd/wails@${REQUIRED_WAILS_VERSION}"
fi

echo "==> Building Windows binary (wails build -platform windows/amd64)"
CC=x86_64-w64-mingw32-gcc CGO_ENABLED=1 wails build -platform windows/amd64
echo "    -> build/bin/opendox-client.exe"

VERSION=$(python3 -c "import json;print(json.load(open('wails.json'))['info']['productVersion'])")
PRODUCT_NAME=$(python3 -c "import json;print(json.load(open('wails.json'))['info']['productName'])")
COMPANY_NAME=$(python3 -c "import json;print(json.load(open('wails.json'))['info']['companyName'])")

echo "==> Building MSI installer (wixl, Version $VERSION)"
mkdir -p build/bin
wixl \
    -D ProductName="$PRODUCT_NAME" \
    -D Manufacturer="$COMPANY_NAME" \
    -D Version="$VERSION" \
    -D ExePath="build/bin/opendox-client.exe" \
    -D SettingsTemplatePath="build/windows/installer/settings.json.template" \
    -D IconPath="build/windows/icon.ico" \
    --arch x64 \
    -o build/bin/opendox-client-installer.msi \
    build/windows/installer/opendox-client.wxs

echo "==> Done."
echo "    -> build/bin/opendox-client-installer.msi"
