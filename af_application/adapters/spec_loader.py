"""
Loads an AFM .md file and returns a validated AgentSpec dict.
"""

import re
import yaml
from pathlib import Path
from schemas.agent_spec import AgentSpec


def load_spec(path: str) -> dict:
    content = Path(path).read_text(encoding="utf-8")

    match = re.match(r"^---\s*\n(.*?)\n---\s*\n", content, re.DOTALL)
    if not match:
        raise ValueError(f"No YAML frontmatter found in {path}")

    frontmatter = yaml.safe_load(match.group(1)) or {}

    body = content[match.end():]
    frontmatter["role"]                 = _extract_section(body, "Role")
    frontmatter["instructions"]         = _extract_section(body, "Instruction") \
                                       or _extract_section(body, "Instructions")
    frontmatter["enforcement"]          = _extract_section(body, "Enforcement")
    frontmatter["json_output_template"] = _extract_output_schema(body)

    spec = AgentSpec(**frontmatter)
    return spec.model_dump()


def _extract_section(body: str, heading: str) -> str | None:
    """Extract text under a # Heading up to the next heading."""
    pattern = rf"#\s+{re.escape(heading)}\s*\n(.*?)(?=\n#\s|\Z)"
    match = re.search(pattern, body, re.DOTALL | re.IGNORECASE)
    if match:
        return match.group(1).strip()
    return None


def _extract_output_schema(body: str) -> str | None:
    """Extract the JSON block under # Output Schema."""
    section = _extract_section(body, "Output Schema")
    if not section:
        return None
    match = re.search(r"```(?:json)?\s*\n(.*?)```", section, re.DOTALL)
    if match:
        return match.group(1).strip()
    return section.strip() or None
