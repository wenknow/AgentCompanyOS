#!/usr/bin/env python3
"""Warn on high-risk tool payloads without blocking normal development."""

from __future__ import annotations

import json
import sys


DANGEROUS_KEYWORDS = [
    "rm -rf",
    "drop database",
    "truncate",
    "kubectl delete",
    "terraform destroy",
    "docker system prune",
    "git push --force",
    "npm publish",
    "deploy prod",
    "production deploy",
    "private key",
    "wallet",
    "transfer",
    "withdraw",
]


def payload_to_text(raw: str) -> str:
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        return raw.lower()
    return json.dumps(parsed, ensure_ascii=False, sort_keys=True).lower()


def main() -> int:
    raw = sys.stdin.read()
    text = payload_to_text(raw)
    matches = [keyword for keyword in DANGEROUS_KEYWORDS if keyword in text]
    if matches:
        print("AgentCompany risk warning: high-risk tool payload detected.", file=sys.stderr)
        print("Matched keywords: " + ", ".join(matches), file=sys.stderr)
        print("Create an approval request or ask for explicit human confirmation before execution.", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

