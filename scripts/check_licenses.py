# openDox - Medical transcription platform
# Copyright (C) 2026  Christopher Abanilla <christopher.abanilla@gmx.de>
# SPDX-FileCopyrightText: 2026 Christopher Abanilla <christopher.abanilla@gmx.de>
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

"""Check the licenses of all shipped dependencies and regenerate THIRD_PARTY_NOTICES.md.

* Go modules: the union of the modules actually linked (``go list -deps``) for
  Linux, Windows and macOS; the license is classified from each module's
  LICENSE/COPYING file (see ``classify``).
* Frontend: production dependencies only (``npm ls --omit=dev``), license from each
  package's package.json. devDependencies never end up in the shipped bundle.

Each license is compared against an allowlist of licenses that may be combined into
an AGPL-3.0-or-later work. Manual-run tool (like ``npm audit``); CI runs it too.
A finding is not necessarily incompatible — it means "look at it by hand".

Usage:
    python3 scripts/check_licenses.py                  # print report, exit 1 on findings
    python3 scripts/check_licenses.py --write-notices  # also (re)write THIRD_PARTY_NOTICES.md
"""

import argparse
import json
import os
import re
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
FRONTEND_DIR = REPO_ROOT / "frontend"
MAIN_MODULE = "opendox-client"
GO_PLATFORMS = ("linux", "windows", "darwin")

# Licenses that can be combined into an AGPL-3.0-or-later work: permissive ones,
# weak-copyleft ones that allow it, public-domain dedications.
ALLOWED_LICENSES = {
    "mit", "mit-0", "bsd", "bsd-2-clause", "bsd-3-clause", "0bsd", "zlib", "isc",
    "apache-2.0", "mpl-2.0", "psf-2.0", "python-2.0", "cc0-1.0", "unlicense",
    "blueoak-1.0.0", "cc-by-4.0", "lgpl-2.1-or-later", "lgpl-3.0-or-later",
    "lgpl-2.1", "lgpl-3.0", "agpl-3.0-or-later", "agpl-3.0", "ofl-1.1",
}

# Modules whose directory holds several licenses (vendored under third_party/): the
# classifier can't pick one, so the declared combination is stated here — keep in
# sync with REUSE.toml and third_party/*/LICENSE*.
LICENSE_OVERRIDES = {
    "github.com/ricknicius99/hid": "BSD-3-Clause AND LGPL-2.1-or-later",
}

LICENSE_FILE_RE = re.compile(r"^(LICEN[CS]E|COPYING|UNLICENSE)(\.(md|txt))?$", re.I)


def classify(text: str) -> str:
    """Best-effort SPDX id from the text of a license file ("UNKNOWN" if unsure)."""
    t = " ".join(text.split())
    low = t.lower()
    if "apache license" in low and "version 2.0" in low:
        return "Apache-2.0"
    if "mozilla public license" in low and "2.0" in low:
        return "MPL-2.0"
    if "gnu affero general public license" in low:
        return "AGPL-3.0-or-later"
    if "gnu lesser general public license" in low:
        return "LGPL-2.1-or-later" if "version 2.1" in low else "LGPL-3.0-or-later"
    if "gnu general public license" in low:
        return "GPL (manual review)"
    if "this is free and unencumbered software released into the public domain" in low:
        return "Unlicense"
    if "redistribution and use in source and binary forms" in low:
        return "BSD-3-Clause" if "neither the name" in low else "BSD-2-Clause"
    if "permission is hereby granted, free of charge" in low:
        return "MIT"
    if "permission to use, copy, modify, and/or distribute this software" in low:
        return "ISC"
    return "UNKNOWN"


def _license_ok(raw: str) -> bool:
    """True if every AND-clause has at least one allowed alternative (OR)."""
    if not raw or raw.strip().upper() == "UNKNOWN":
        return False
    expr = raw.replace("(", " ").replace(")", " ")
    for and_clause in re.split(r"\s+AND\s+|;", expr):
        alternatives = [a.strip().lower() for a in re.split(r"\s+OR\s+", and_clause) if a.strip()]
        if not any(a in ALLOWED_LICENSES for a in alternatives):
            return False
    return True


def get_go_licenses() -> list[dict]:
    """Union of the Go modules linked into the app on Linux, Windows and macOS."""
    modules: dict[tuple[str, str], str] = {}
    for goos in GO_PLATFORMS:
        env = {**os.environ, "GOOS": goos, "CGO_ENABLED": "1"}
        out = subprocess.run(
            ["go", "list", "-deps", "-f", "{{with .Module}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}", "."],
            cwd=REPO_ROOT, env=env, capture_output=True, text=True, check=True,
        ).stdout
        for line in out.splitlines():
            path, _, rest = line.partition("|")
            version, _, directory = rest.partition("|")
            if path and path != MAIN_MODULE:
                modules[(path, version or "local")] = directory

    packages = []
    for (path, version), directory in sorted(modules.items()):
        if path in LICENSE_OVERRIDES:
            license_id = LICENSE_OVERRIDES[path]
        else:
            files = [f for f in Path(directory).iterdir() if f.is_file() and LICENSE_FILE_RE.match(f.name)]
            license_id = classify(files[0].read_text(errors="replace")) if files else "UNKNOWN"
        packages.append({"name": path, "version": version, "license": license_id})
    return packages


def get_frontend_licenses() -> list[dict]:
    """Production npm dependencies with the license from their package.json."""
    out = subprocess.run(
        ["npm", "ls", "--omit=dev", "--all", "--parseable"],
        cwd=FRONTEND_DIR, capture_output=True, text=True,
    ).stdout  # npm ls exits non-zero on e.g. extraneous packages; the list is still valid
    own = FRONTEND_DIR.resolve()
    packages = {}
    for line in out.splitlines():
        directory = Path(line.strip())
        manifest = directory / "package.json"
        if not line.strip() or directory.resolve() == own or not manifest.is_file():
            continue
        data = json.loads(manifest.read_text(encoding="utf-8"))
        license_field = data.get("license") or data.get("licenses") or "UNKNOWN"
        if isinstance(license_field, dict):
            license_field = license_field.get("type", "UNKNOWN")
        elif isinstance(license_field, list):
            license_field = " OR ".join(item.get("type", "UNKNOWN") for item in license_field)
        packages[(data["name"], data.get("version", "?"))] = str(license_field)
    return [{"name": n, "version": v, "license": lic} for (n, v), lic in sorted(packages.items())]


def write_third_party_notices(go: list[dict], frontend: list[dict]) -> None:
    """Regenerate THIRD_PARTY_NOTICES.md (generated, not hand-maintained)."""
    lines = [
        "# Third-Party Notices",
        "",
        "openDox Client is licensed under the GNU Affero General Public License",
        "v3.0-or-later (see `LICENSE`). It depends on the third-party components",
        "listed below, each under its own license. This file is generated by",
        "`python3 scripts/check_licenses.py --write-notices` — do not hand-edit.",
        "",
        f"## Go modules ({len(go)})",
        "",
        "Union of the modules linked on Linux, Windows and macOS. `hid` and `hotkey` are",
        "vendored under `third_party/` (see `go.mod`); `hid` bundles hidapi (BSD-3-Clause)",
        "and libusb (LGPL-2.1-or-later, dynamically linked on Linux).",
        "",
        "| Module | Version | License |",
        "|---|---|---|",
    ]
    lines += [f"| {p['name']} | {p['version']} | {p['license']} |" for p in go]
    lines += [
        "",
        f"## Frontend (npm, production, {len(frontend)} packages)",
        "",
        "| Package | Version | License |",
        "|---|---|---|",
    ]
    lines += [f"| {p['name']} | {p['version']} | {p['license']} |" for p in frontend]
    lines += [
        "",
        "## Embedded assets",
        "",
        "| Asset | License |",
        "|---|---|",
        "| Nunito font (`frontend/src/assets/fonts`) | OFL-1.1 (© The Nunito Project Authors) |",
        "| Wails JS runtime (`frontend/wailsjs/runtime`) | MIT (© Lea Anthony) |",
        "",
        "## Not part of this repository",
        "",
        "The Windows NSIS installer can download Microsoft's WebView2 bootstrapper at build",
        "time (Wails). It is a Microsoft redistributable under Microsoft's own terms and is",
        "deliberately not checked in.",
        "",
    ]
    (REPO_ROOT / "THIRD_PARTY_NOTICES.md").write_text("\n".join(lines), encoding="utf-8")


def main() -> int:
    """Run both license checks and print a report."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write-notices", action="store_true",
                        help="Regenerate THIRD_PARTY_NOTICES.md from the current dependency set")
    args = parser.parse_args()

    go = get_go_licenses()
    frontend = get_frontend_licenses()

    if args.write_notices:
        write_third_party_notices(go, frontend)
        print(f"Wrote THIRD_PARTY_NOTICES.md ({len(go)} Go modules + {len(frontend)} npm packages)")

    flagged = [("go", p) for p in go if not _license_ok(p["license"])]
    flagged += [("npm", p) for p in frontend if not _license_ok(p["license"])]
    if not flagged:
        print(f"OK: {len(go)} Go + {len(frontend)} npm licenses are all AGPL-3.0-compatible.")
        return 0

    print("Licenses needing manual review (not necessarily incompatible, just not on the allowlist):")
    for kind, p in flagged:
        print(f"  [{kind}] {p['name']} {p['version']}: {p['license']}")
    return 1


if __name__ == "__main__":
    sys.exit(main())
