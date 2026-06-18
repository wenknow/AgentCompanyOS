#!/usr/bin/env python3
"""Initialize a project with AgentCompanyOS Codex standards."""

from __future__ import annotations

import argparse
import shutil
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
TEMPLATES = ROOT / "templates"


def copy_file(src: Path, dst: Path, force: bool) -> bool:
    if dst.exists() and not force:
        return False
    dst.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(src, dst)
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate AGENTS.md for a target project.")
    parser.add_argument("target", nargs="?", default=".", help="Target project directory.")
    parser.add_argument("--force", action="store_true", help="Overwrite existing generated files.")
    parser.add_argument("--readme", action="store_true", help="Also copy README.template.md to README.md if allowed.")
    args = parser.parse_args()

    target = Path(args.target).resolve()
    target.mkdir(parents=True, exist_ok=True)

    agents_created = copy_file(TEMPLATES / "AGENTS.md", target / "AGENTS.md", args.force)
    print(("created" if agents_created else "skipped") + f" {target / 'AGENTS.md'}")

    if args.readme:
        readme_created = copy_file(TEMPLATES / "README.template.md", target / "README.md", args.force)
        print(("created" if readme_created else "skipped") + f" {target / 'README.md'}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())

