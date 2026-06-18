#!/usr/bin/env python3
"""Print AgentCompanyOS company standards summary for Codex sessions."""

from __future__ import annotations


def main() -> int:
    print("AgentCompanyOS standards loaded.")
    print("- Safe by default: prefer read-only, draft, simulation, and approval flows.")
    print("- High-risk actions require explicit Founder approval.")
    print("- Do not deploy production, merge code, publish content, operate funds, or change risk rules automatically.")
    print("- Do not read, print, or store secrets, tokens, wallet private keys, or credentials.")
    print("- Keep significant actions auditable.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

