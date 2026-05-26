"""
Skills adapter — loads local skill definitions and executes them.

A local skill is a markdown file in the skills directory.
Each .md file is a skill with a name (filename without extension)
and content (the skill instructions/workflow).

Skills are surfaced to the LLM as context in the system prompt
so it knows what structured workflows are available.
"""

import os
from pathlib import Path


def load_local_skills(base_path: str, spec_dir: str) -> list[dict]:
    """
    Load all .md skill files from a local skills directory.

    Args:
        base_path: path from the spec (e.g. "./skills")
        spec_dir:  directory of the .md spec file (used to resolve relative paths)

    Returns:
        List of {"name": str, "content": str} dicts
    """
    # Resolve relative path against the spec file's directory
    if base_path.startswith("./") or base_path.startswith(".\\"):
        resolved = os.path.join(spec_dir, base_path)
    else:
        resolved = base_path

    resolved = os.path.normpath(resolved)

    if not os.path.isdir(resolved):
        return []

    skills = []
    for md_file in sorted(Path(resolved).glob("*.md")):
        try:
            content = md_file.read_text(encoding="utf-8").strip()
            skills.append({
                "name":    md_file.stem,
                "content": content,
            })
        except Exception:
            pass

    return skills


def build_skills_context(agent_spec: dict, spec_path: str) -> str | None:
    """
    Build a system prompt section describing available skills.
    Returns None if no skills are defined.
    """
    skill_defs = agent_spec.get("skills") or []
    if not skill_defs:
        return None

    spec_dir = os.path.dirname(os.path.abspath(spec_path))
    all_skills = []

    for skill_def in skill_defs:
        skill_type = skill_def.get("type", "local")
        if skill_type == "local":
            path = skill_def.get("path", "./skills")
            loaded = load_local_skills(path, spec_dir)
            all_skills.extend(loaded)

    if not all_skills:
        return None

    lines = ["## Available Skills",
             "You have access to the following structured workflows (skills).",
             "Activate the relevant skill when the user's request matches its purpose.\n"]

    for skill in all_skills:
        lines.append(f"### Skill: {skill['name']}")
        lines.append(skill["content"])
        lines.append("")

    return "\n".join(lines)
