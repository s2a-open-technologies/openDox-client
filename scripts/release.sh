#!/usr/bin/env bash
# scripts/release.sh — vorhandene Builds als GitHub-Release hochladen
#
# Der openDox-Server (Admin → Clients) verlinkt auf
#   https://github.com/daemolition/openDox-client/releases/latest/download/<Dateiname>
# Die Dateinamen hier sind deshalb FEST — ändern nur zusammen mit
# _CLIENT_DOWNLOAD_FILES in backend/views/settings_api/settings_routes.py (Server-Repo).
#
# Ablauf:
#   Linux:  ./scripts/build-windows-msi.sh && ./scripts/compile.sh && scripts/release.sh 0.8.0
#   macOS:  ./scripts/build-macos.sh && scripts/release.sh 0.8.0   (lädt ins selbe Release nach)
#
# Lädt nur hoch, was unter build/bin/ liegt; bestehende Dateien im Release werden ersetzt.
# Normalerweise legt .github/workflows/release.yml das Release beim Tag-Push an (mit den
# CHANGELOG-Notizen) und baut Windows-MSI/-EXE und das Linux-Binary UNSIGNIERT in der CI.
# Dieses Skript ersetzt sie durch lokal gebaute, signierte Builds (gleiche Dateinamen) und
# lädt das macOS-ZIP hoch, das sich in der CI nicht bauen lässt.
# Builds vorher signieren (Smart App Control, siehe build-macos.sh / DEPLOYMENT.md).
#
# Voraussetzung: gh CLI (sudo apt install gh / brew install gh), einmalig `gh auth login`.
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ $# -ne 1 ]]; then
    echo "Aufruf: $0 <version>   (z.B. 0.8.0)" >&2
    exit 1
fi
VERSION="${1#v}"
TAG="v$VERSION"

if ! command -v gh >/dev/null 2>&1; then
    echo "Fehler: gh nicht gefunden (sudo apt install gh, danach gh auth login)." >&2
    exit 1
fi

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
ASSETS=()

add_asset() { # <quelle> <fester-name>
    if [[ -f "$1" ]]; then
        cp "$1" "$STAGE/$2"
        ASSETS+=("$STAGE/$2")
        echo "    + $2  (aus $1)"
    fi
}

echo "==> Sammle Builds für $TAG"
add_asset build/bin/opendox-client-installer.msi opendox-client-installer.msi
add_asset build/bin/opendox-client-amd64-installer.exe opendox-client-amd64-installer.exe
if [[ -d build/bin/opendox-client.app ]] && command -v ditto >/dev/null 2>&1; then
    # ditto erhält Signatur und erweiterte Attribute des Bundles (zip -r tut das nicht).
    ditto -c -k --keepParent build/bin/opendox-client.app "$STAGE/opendox-client-macos.zip"
    ASSETS+=("$STAGE/opendox-client-macos.zip")
    echo "    + opendox-client-macos.zip  (aus build/bin/opendox-client.app)"
fi

if [[ ${#ASSETS[@]} -eq 0 ]]; then
    echo "Fehler: keine Builds unter build/bin/ gefunden." >&2
    exit 1
fi

if gh release view "$TAG" >/dev/null 2>&1; then
    echo "==> Release $TAG existiert, lade hoch (ersetzt gleichnamige Dateien)"
    gh release upload "$TAG" "${ASSETS[@]}" --clobber
else
    echo "==> Erstelle Release $TAG"
    # Notizen aus CHANGELOG.md (dieselbe Extraktion wie .github/workflows/release.yml);
    # fehlt der Abschnitt, lässt GitHub sie aus den Commits erzeugen.
    NOTES="$STAGE/release_notes.md"
    awk -v ver="$VERSION" '
        /^## \[/ { if (found) exit; if (index($0, "[" ver "]") > 0) { found=1 } next }
        found { print }
    ' CHANGELOG.md > "$NOTES"
    if [[ -s "$NOTES" ]]; then
        NOTES_ARGS=(--notes-file "$NOTES")
    else
        NOTES_ARGS=(--generate-notes)
    fi
    gh release create "$TAG" "${ASSETS[@]}" --title "$TAG" "${NOTES_ARGS[@]}"
fi
echo "==> Fertig: $(gh release view "$TAG" --json url -q .url)"
