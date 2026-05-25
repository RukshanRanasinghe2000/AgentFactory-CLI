import os
import re

from schemas.runtime_state import RuntimeState
from adapters.runtime_adapter import execute_http_tool, execute_stdio_tool


def _resolve_env(value: str) -> str:
    def replacer(match):
        return os.environ.get(match.group(1), "")
    return re.sub(r"\$\{env:([^}]+)\}", replacer, value or "")


def _find_tool_schema(tool_schemas: list, tool_name: str) -> dict | None:
    """Find the full tool schema (with routing metadata) by function name."""
    for s in tool_schemas:
        if s.get("function", {}).get("name") == tool_name:
            return s
    return None


def _find_http_tool(agent_spec: dict, tool_name: str) -> dict | None:
    """Find an HTTP tool definition in the spec by name."""
    mcp_tools = (agent_spec.get("tools") or {}).get("mcp") or []
    for tool in mcp_tools:
        if tool.get("name") == tool_name:
            return tool
    return None


def _build_http_url(tool_def: dict, arguments: dict) -> str:
    url = tool_def.get("transport", {}).get("url", "")
    for key, value in arguments.items():
        url = url.replace(f"{{{key}}}", str(value))
    raw_key = (tool_def.get("authentication") or {}).get("api_key", "")
    url = url.replace("{API_KEY}", _resolve_env(raw_key))
    return url


def tools_node(state: RuntimeState) -> RuntimeState:
    agent_spec   = state.get("agent_spec", {})
    tool_calls   = state.get("tool_calls") or []
    tool_schemas = state.get("_tool_schemas") or []
    tool_results = []

    for tool_call in tool_calls:
        tool_name = tool_call.get("tool")
        arguments = tool_call.get("arguments", {})

        # Look up routing metadata from the schema built in chat_node
        schema = _find_tool_schema(tool_schemas, tool_name)

        if schema:
            transport = schema.get("_transport", {})
            transport_type = transport.get("type", "http")
        else:
            # Fallback: look up directly in spec (HTTP tools)
            tool_def = _find_http_tool(agent_spec, tool_name)
            if not tool_def:
                tool_results.append({
                    "id": tool_call.get("id", tool_name),
                    "tool": tool_name,
                    "success": False,
                    "error": "Tool not found",
                })
                continue
            transport = tool_def.get("transport", {})
            transport_type = transport.get("type", "http")

        if transport_type == "stdio":
            result = execute_stdio_tool(
                transport=transport,
                tool_name=tool_name,
                arguments=arguments,
            )
        else:
            # HTTP — need the full tool def for URL building
            tool_def = _find_http_tool(agent_spec, schema.get("_mcp_server") if schema else tool_name)
            url = _build_http_url(tool_def or {}, arguments)
            result = execute_http_tool(url)

        tool_results.append({
            "id":      tool_call.get("id", tool_name),
            "tool":    tool_name,
            "success": result.get("success", False),
            "result":  result,
        })

    state["tool_results"] = tool_results
    return state
