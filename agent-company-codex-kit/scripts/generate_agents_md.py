#!/usr/bin/env python3
"""Print the default AgentCompanyOS AGENTS.md template."""

from __future__ import annotations

from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def main() -> int:
    print((ROOT / "templates" / "AGENTS.md").read_text(encoding="utf-8"), end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

