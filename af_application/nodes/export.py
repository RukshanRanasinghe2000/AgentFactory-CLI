from pathlib import Path

import yaml

from schemas.state import AgentState


EXPORT_DIR = Path("exports")
EXPORT_DIR.mkdir(exist_ok=True)


def export_node(state: AgentState):
    """
    Export AgentSpec into agent.md format.
    """

    spec = state.get("agent_spec")

    if not spec:
        state["validation_errors"] = [
            "No validated spec available for export"
        ]

        return state

    agent_name = spec.get("name", "agent")

    file_name = agent_name.lower().replace(" ", "-") + ".md"

    output_path = EXPORT_DIR / file_name

    yaml_content = yaml.dump(
        spec,
        sort_keys=False,
        allow_unicode=True
    )

    markdown_content = f"---\n{yaml_content}---\n"

    with open(output_path, "w", encoding="utf-8") as f:
        f.write(markdown_content)

    state["exported_file"] = str(output_path)

    return state