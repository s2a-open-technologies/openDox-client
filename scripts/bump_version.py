# openDox - Medical transcription platform
# Copyright (C) 2026  S2A Open Technologies gGmbH
# SPDX-FileCopyrightText: 2026 S2A Open Technologies gGmbH
# SPDX-License-Identifier: AGPL-3.0-or-later
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

"""Bump the single canonical client version.

``wails.json`` (``info.productVersion``) is the source of truth: Wails embeds it
into the Windows resource / macOS Info.plist, and ``build-windows-msi.sh`` reads it
for the MSI. ``frontend/package.json`` (+ lockfile) is kept in sync.

Usage:
    python3 scripts/bump_version.py --current       # print current version
    python3 scripts/bump_version.py patch|minor|major
    python3 scripts/bump_version.py 1.2.3            # explicit version
    python3 scripts/bump_version.py patch --commit   # also git add + commit + tag

Moves CHANGELOG.md's "[Unreleased]" section under the new version heading.
Pushing the resulting ``vX.Y.Z`` tag triggers .github/workflows/release.yml,
which publishes that section as the GitHub Release notes; the installers are
attached afterwards with scripts/release.sh.
"""

import argparse
import re
import subprocess
import sys
from datetime import date
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent
WAILS_JSON = PROJECT_ROOT / "wails.json"
FRONTEND_DIR = PROJECT_ROOT / "frontend"
PACKAGE_JSON = FRONTEND_DIR / "package.json"
PACKAGE_LOCK = FRONTEND_DIR / "package-lock.json"
CHANGELOG = PROJECT_ROOT / "CHANGELOG.md"
REPO_URL = "https://github.com/daemolition/openDox-client"

VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
WAILS_VERSION_RE = re.compile(r'^(\s*"productVersion":\s*)"(.*)"(,?)$', re.MULTILINE)


def read_current_version() -> str:
    """Read the current client version from wails.json (info.productVersion)."""
    match = WAILS_VERSION_RE.search(WAILS_JSON.read_text(encoding="utf-8"))
    if not match:
        raise ValueError("info.productVersion not found in wails.json")
    return match.group(2)


def compute_new_version(current: str, target: str) -> str:
    """Compute the new version from a bump component or an explicit value.

    Raises:
        ValueError: if target is neither major/minor/patch nor "X.Y.Z".
    """
    if VERSION_RE.match(target):
        return target
    if not VERSION_RE.match(current):
        raise ValueError(f"current version {current!r} is not X.Y.Z")

    major, minor, patch = (int(part) for part in current.split("."))
    if target == "major":
        return f"{major + 1}.0.0"
    if target == "minor":
        return f"{major}.{minor + 1}.0"
    if target == "patch":
        return f"{major}.{minor}.{patch + 1}"
    raise ValueError(f"Invalid bump target: {target!r}")


def write_wails_version(new_version: str) -> None:
    """Replace info.productVersion in wails.json in place (keeps the formatting)."""
    text = WAILS_JSON.read_text(encoding="utf-8")
    updated = WAILS_VERSION_RE.sub(rf'\1"{new_version}"\3', text, count=1)
    WAILS_JSON.write_text(updated, encoding="utf-8")


def write_package_json_version(new_version: str) -> None:
    """Replace the top-level version field in frontend/package.json in place."""
    text = PACKAGE_JSON.read_text(encoding="utf-8")
    updated = re.sub(
        r'^(\s*)"version": ".*",$',
        rf'\1"version": "{new_version}",',
        text,
        count=1,
        flags=re.MULTILINE,
    )
    PACKAGE_JSON.write_text(updated, encoding="utf-8")


def resync_package_lock() -> None:
    """Resync package-lock.json's embedded version fields via npm."""
    subprocess.run(
        ["npm", "install", "--package-lock-only", "--ignore-scripts"],
        cwd=FRONTEND_DIR,
        check=True,
    )


def update_changelog(current: str, new_version: str) -> bool:
    """Move the "[Unreleased]" CHANGELOG section under a new version heading.

    Renames "## [Unreleased]" to "## [<new_version>] — <today>", inserts a fresh,
    empty "## [Unreleased]" above it and rewrites the compare links at the bottom.
    No-op (returns False) if "[Unreleased]" has no bullets yet: bumping the version
    is no reason to create an empty release section.
    """
    text = CHANGELOG.read_text(encoding="utf-8")

    match = re.search(r"^## \[Unreleased\]\n(.*?)(?=^## \[|\Z)", text, re.S | re.M)
    if not match or "\n- " not in ("\n" + match.group(1)):
        return False

    today = date.today().isoformat()
    heading = f"## [Unreleased]\n\n## [{new_version}] — {today}\n"
    updated = text[: match.start()] + heading + match.group(1) + text[match.end() :]

    updated = re.sub(
        rf"^\[Unreleased\]: {re.escape(REPO_URL)}/compare/v.*\.\.\.HEAD$",
        f"[Unreleased]: {REPO_URL}/compare/v{new_version}...HEAD\n"
        f"[{new_version}]: {REPO_URL}/compare/v{current}...v{new_version}",
        updated,
        count=1,
        flags=re.MULTILINE,
    )

    CHANGELOG.write_text(updated, encoding="utf-8")
    return True


def commit_and_tag(new_version: str, changelog_updated: bool) -> None:
    """Commit the version bump and create an annotated git tag."""
    files = [str(WAILS_JSON), str(PACKAGE_JSON), str(PACKAGE_LOCK)]
    if changelog_updated:
        files.append(str(CHANGELOG))
    subprocess.run(["git", "add", *files], cwd=PROJECT_ROOT, check=True)
    subprocess.run(
        ["git", "commit", "-m", f"chore: bump version to v{new_version}"],
        cwd=PROJECT_ROOT,
        check=True,
    )
    subprocess.run(
        ["git", "tag", "-a", f"v{new_version}", "-m", f"v{new_version}"],
        cwd=PROJECT_ROOT,
        check=True,
    )


def main() -> None:
    """Parse CLI arguments and bump the client version accordingly."""
    parser = argparse.ArgumentParser(
        description="Bump the client version across wails.json and frontend/package.json."
    )
    parser.add_argument(
        "target",
        nargs="?",
        help='"major", "minor", "patch", or an explicit "X.Y.Z" version',
    )
    parser.add_argument(
        "--current", action="store_true", help="print current version and exit"
    )
    parser.add_argument(
        "--commit",
        action="store_true",
        help="also git add + commit + tag the bumped files",
    )
    args = parser.parse_args()

    current = read_current_version()

    if args.current:
        print(current)
        return

    if not args.target:
        parser.error("target is required unless --current is given")

    new_version = compute_new_version(current, args.target)

    write_wails_version(new_version)
    write_package_json_version(new_version)
    resync_package_lock()
    changelog_updated = update_changelog(current, new_version)
    if not changelog_updated:
        print(
            "Warning: CHANGELOG.md's [Unreleased] section is empty — "
            "no version section created.",
            file=sys.stderr,
        )

    if args.commit:
        commit_and_tag(new_version, changelog_updated)

    print(f"{current} -> {new_version}")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, subprocess.CalledProcessError) as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
