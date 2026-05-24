import os
import re
from schemas.runtime_state import RuntimeState
from adapters.runtime_adapter import execute_tool_call


def find_tool(agent_spec, tool_name):
    mcp_tools = (agent_spec.get("tools") or {}).get("mcp") or []
    for tool in mcp_tools:
        if tool.get("name") == tool_name:
            return tool
    return None


def resolve_env(value: str) -> str:
    """Resolve ${env:VAR_NAME} patterns to their environment values."""
    def replacer(match):
        var = match.group(1)
        return os.environ.get(var, "")
    return re.sub(r"\$\{env:([^}]+)\}", replacer, value or "")


def build_tool_url(tool_definition, arguments):
    """
    Build the final URL by:
    1. Replacing {PLACEHOLDER} with LLM-provided arguments
    2. Replacing {API_KEY} with the tool's resolved auth key
    3. Resolving any ${env:VAR} references
    """
    url = tool_definition.get("transport", {}).get("url", "")

    # Inject LLM arguments
    for key, value in arguments.items():
        url = url.replace(f"{{{key}}}", str(value))

    # Inject auth API key into {API_KEY} placeholder
    raw_api_key = (
        tool_definition
        .get("authentication", {})
        .get("api_key", "")
    )
    resolved_key = resolve_env(raw_api_key)
    url = url.replace("{API_KEY}", resolved_key)

    return url


def tools_node(state: RuntimeState):
    """
    Execute dynamically requested tools.
    """

    agent_spec = state.get("agent_spec", {})
    tool_calls = state.get("tool_calls") or []

    tool_results = []

    for tool_call in tool_calls:

        tool_name = tool_call.get("tool")
        arguments = tool_call.get("arguments", {})

        tool_definition = find_tool(
            agent_spec,
            tool_name
        )

        if not tool_definition:

            tool_results.append({
                "tool": tool_name,
                "success": False,
                "error": "Tool not found"
            })

            continue

        url = build_tool_url(
            tool_definition,
            arguments
        )

        result = execute_tool_call(url)

        tool_results.append({
            "id":      tool_call.get("id", tool_name),
            "tool":    tool_name,
            "success": result.get("success", False),
            "result":  result
        })

    state["tool_results"] = tool_results

    return state