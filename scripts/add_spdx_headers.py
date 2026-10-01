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

"""Add REUSE-compliant SPDX short-form tags to every file that already has the header.

Inserts the two SPDX lines right after the existing
"Copyright (C) <year> Christopher Abanilla ..." line of the full-text AGPL header,
whatever the comment style (Go ``//``, TS/TSX ``/* */``, Python ``#``): everything
in front of "Copyright (C)" on that line is reused as the prefix for the new lines,
so no per-extension special-casing is needed.

Files without any header are not touched (a first header is an editorial decision),
and neither are vendored/generated paths (``third_party/``, ``frontend/wailsjs/``,
...) — REUSE.toml covers those in bulk. Re-run after adding new headered files.

Usage:
    python3 scripts/add_spdx_headers.py            # apply
    python3 scripts/add_spdx_headers.py --dry-run  # list files that would change
"""

import argparse
import re
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent

INCLUDE_EXTENSIONS = {".go", ".ts", ".tsx", ".css", ".py"}

# Path prefixes (relative, POSIX) that are skipped: vendored, generated, build output.
EXCLUDE_PREFIXES = (
    ".git/",
    "third_party/",
    "frontend/node_modules/",
    "frontend/dist/",
    "frontend/wailsjs/",
    "build/bin/",
)

COPYRIGHT_RE = re.compile(r"Copyright \(C\) (\d{4})\s+Christopher Abanilla")
# Split so this file's own source doesn't itself look like an (invalid, templated)
# SPDX tag to `reuse lint`'s line scanner or to this script's "already done" check.
SPDX_MARKER = "SPDX-" + "License-Identifier"
SPDX_LICENSE_EXPR = "AGPL-3.0-" + "or-later"


def _iter_candidate_files():
    for path in REPO_ROOT.rglob("*"):
        if not path.is_file():
            continue
        rel = path.relative_to(REPO_ROOT).as_posix()
        if rel.startswith(EXCLUDE_PREFIXES):
            continue
        if path.suffix in INCLUDE_EXTENSIONS:
            yield path


def add_spdx_to_file(path: Path) -> bool:
    """Insert SPDX tags after the Copyright line. Returns True if changed."""
    text = path.read_text(encoding="utf-8")
    if SPDX_MARKER in text:
        return False  # already done

    lines = text.splitlines(keepends=True)
    for i, line in enumerate(lines):
        match = COPYRIGHT_RE.search(line)
        if not match:
            continue
        year = match.group(1)
        prefix = line[: match.start()]
        newline = "\n" if line.endswith("\n") else ""
        spdx_lines = [
            f"{prefix}SPDX-FileCopyrightText: {year} Christopher Abanilla "
            f"<christopher.abanilla@gmx.de>{newline}",
            f"{prefix}{SPDX_MARKER}: {SPDX_LICENSE_EXPR}{newline}",
        ]
        lines[i + 1 : i + 1] = spdx_lines
        path.write_text("".join(lines), encoding="utf-8")
        return True
    return False


def main() -> int:
    """Walk the repo and add SPDX tags to every already-headered file."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    changed = []
    no_header = []
    for path in sorted(_iter_candidate_files()):
        rel = path.relative_to(REPO_ROOT)
        text = path.read_text(encoding="utf-8")
        if SPDX_MARKER in text:
            continue
        if not COPYRIGHT_RE.search(text):
            no_header.append(rel)
            continue
        if args.dry_run or add_spdx_to_file(path):
            changed.append(rel)

    for rel in changed:
        print(f"{'would update' if args.dry_run else 'updated'}: {rel}")
    if no_header:
        print(f"\n{len(no_header)} file(s) without a Copyright line (need a header by hand):")
        for rel in no_header:
            print(f"  {rel}")

    print(f"\n{len(changed)} file(s) {'would be ' if args.dry_run else ''}updated.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
