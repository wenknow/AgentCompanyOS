#!/usr/bin/env python3
"""Validate AgentCompany Codex Kit required files."""

from __future__ import annotations

import json
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]

REQUIRED_REFERENCES = [
    "company-policy.md",
    "engineering-standard.md",
    "golang-standard.md",
    "security-standard.md",
    "risk-control-standard.md",
    "approval-policy.md",
    "project-workflow.md",
    "mcp-policy.md",
    "agent-company-os-architecture.md",
]

REQUIRED_SKILLS = [
    "company-go-backend",
    "company-architecture-review",
    "company-risk-review",
    "company-prd-to-tasks",
    "company-code-review",
    "company-test-and-lint",
    "agent-company-os-dev",
]


def check_json(path: Path) -> bool:
    try:
        json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        print(f"invalid json {path}: {exc}")
        return False
    print(f"ok json {path}")
    return True


def check_exists(path: Path) -> bool:
    if path.exists():
        print(f"ok {path}")
        return True
    print(f"missing {path}")
    return False


def skill_has_frontmatter(path: Path) -> bool:
    if not check_exists(path):
        return False
    text = path.read_text(encoding="utf-8")
    ok = text.startswith("---\n") and "\nname:" in text and "\ndescription:" in text
    print(("ok" if ok else "missing frontmatter") + f" {path}")
    return ok


def main() -> int:
    checks = []
    checks.append(check_json(ROOT / ".codex-plugin" / "plugin.json"))
    checks.append(check_json(ROOT / ".mcp.json"))
    checks.append(check_json(ROOT / "hooks" / "hooks.json"))
    checks.extend(check_exists(ROOT / "references" / name) for name in REQUIRED_REFERENCES)
    checks.extend(skill_has_frontmatter(ROOT / "skills" / name / "SKILL.md") for name in REQUIRED_SKILLS)
    checks.append(check_exists(ROOT / "README.md"))
    checks.append(check_exists(ROOT / "marketplace.json"))
    if all(checks):
        print("validation passed")
        return 0
    print("validation failed")
    return 1


if __name__ == "__main__":
    raise SystemExit(main())

