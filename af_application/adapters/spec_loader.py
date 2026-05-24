"""
Loads an AFM .md file and returns a validated AgentSpec dict.

Format expected:
---
<YAML frontmatter>
---
# Role
...
# Instructions
...
"""

import re
import yaml
from pathlib import Path
from schemas.agent_spec import AgentSpec


def load_spec(path: str) -> dict:
    """
    Parse an AFM .md file into a validated AgentSpec dict.
    Raises ValueError if the file has no frontmatter or fails validation.
    """
    content = Path(path).read_text(encoding="utf-8")

    # Extract YAML frontmatter between the first two --- markers
    match = re.match(r"^---\s*\n(.*?)\n---\s*\n", content, re.DOTALL)
    if not match:
        raise ValueError(f"No YAML frontmatter found in {path}")

    frontmatter = yaml.safe_load(match.group(1)) or {}

    # Extract markdown body sections after the closing ---
    body = content[match.end():]
    frontmatter["role"]         = _extract_section(body, "Role")
    frontmatter["instructions"] = _extract_section(body, "Instruction") \
                                or _extract_section(body, "Instructions")
    frontmatter["enforcement"]  = _extract_section(body, "Enforcement")

    # Validate through Pydantic — returns a clean dict
    spec = AgentSpec(**frontmatter)
    return spec.model_dump()


def _extract_section(body: str, heading: str) -> str | None:
    """
    Extract the text under a markdown # Heading, up to the next heading.
    Returns None if the heading is not found.
    """
    pattern = rf"#\s+{re.escape(heading)}\s*\n(.*?)(?=\n#\s|\Z)"
    match = re.search(pattern, body, re.DOTALL | re.IGNORECASE)
    if match:
        return match.group(1).strip()
    return None
