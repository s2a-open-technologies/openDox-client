#!/usr/bin/env bash
# build-macos.sh — wails build + Signierung
#
# Warum: Bei jedem `wails build` entsteht ein neues Binary mit anderem Hash. Mit
# reiner ad-hoc-Signatur (`codesign --sign -`) hält macOS TCC (Mikrofon/Bedienhilfen/
# Eingabeüberwachung) das NICHT zuverlässig auseinander — empirisch bestätigt (macOS
# 26.4/arm64): der Bedienhilfen-Schalter blieb in den Systemeinstellungen sichtbar AN,
# aber AXIsProcessTrusted() im Prozess selbst lieferte trotzdem dauerhaft false. Ein
# selbstsigniertes lokales Zertifikat wurde ebenfalls ausprobiert und verworfen:
# `spctl` (Gatekeeper) lehnt damit signierte Builds explizit als "von nicht
# vertrauter Stelle" ab, ohne erkennbaren Vorteil gegenüber ad-hoc.
#
# Einzige Methode, die tatsächlich hielt: ein ECHTES, von Apple ausgestelltes
# Zertifikat. Dafür reicht das KOSTENLOSE "Apple Development"-Zertifikat über
# Xcode (kein 99€-Program nötig):
#   Xcode → Settings (Cmd+,) → Accounts → "+" → Apple ID anmelden → Team wählen
#   → "Manage Certificates…" → "+" → "Apple Development"
# Falls `security find-identity -v -p codesigning` danach trotzdem "0 valid
# identities" zeigt: fehlt wahrscheinlich die aktuelle Apple-WWDR-Zwischenzertifikat-
# Kette im Schlüsselbund (bei uns lag nur eine seit 2023 abgelaufene G3-Version vor).
# Aktuelle nachladen:
#   curl -sL -o /tmp/AppleWWDRCAG3.cer https://www.apple.com/certificateauthority/AppleWWDRCAG3.cer
#   security import /tmp/AppleWWDRCAG3.cer -k ~/Library/Keychains/login.keychain-db
#
# Dieses Skript sucht automatisch nach einer "Apple Development"-Identität im
# Schlüsselbund und signiert damit; ohne eine solche fällt es auf ad-hoc zurück
# (funktioniert dann, aber ohne die TCC-Persistenz-Garantie oben).
#
# Aufruf statt `wails build`:
#   ./build-macos.sh          # Standard (arm64)
#   ./build-macos.sh --debug  # Debug-Build
set -euo pipefail

cd "$(dirname "$0")"

echo "==> wails build"
wails build "$@"

APP="build/bin/opendox-client.app"
if [[ ! -d "$APP" ]]; then
  echo "Fehler: $APP nicht gefunden" >&2
  exit 1
fi

IDENTITY=$(security find-identity -v -p codesigning 2>/dev/null | grep -o '"Apple Development[^"]*"' | head -1 | tr -d '"' || true)

if [[ -n "$IDENTITY" ]]; then
  echo "==> Signierung von $APP mit Apple-Development-Zertifikat ($IDENTITY)"
  codesign --force --deep --sign "$IDENTITY" "$APP"
else
  echo "==> Kein Apple-Development-Zertifikat gefunden — ad-hoc-Signierung von $APP"
  echo "    (TCC-Berechtigungen persistieren dann möglicherweise nicht zuverlässig — siehe Kommentar oben)"
  codesign --force --deep --sign - "$APP"
fi

echo "==> Bundle-Identifier verifizieren"
/usr/libexec/PlistBuddy -c "Print :CFBundleIdentifier" "$APP/Contents/Info.plist"
echo "Fertig: $APP"

