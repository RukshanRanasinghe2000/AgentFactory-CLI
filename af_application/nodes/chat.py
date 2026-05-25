import os
from schemas.runtime_state import RuntimeState
from providers.openai import generate_with_tools
from adapters.runtime_adapter import list_stdio_tools


def _clean_schema(schema: dict) -> dict:
    """Remove keys that Groq/OpenAI reject from JSON schemas."""
    if not isinstance(schema, dict):
        return schema
    cleaned = {}
    for k, v in schema.items():
        if k in ("$schema", "additionalProperties", "outputSchema",
                 "annotations", "execution", "title"):
            continue
        if isinstance(v, dict):
            cleaned[k] = _clean_schema(v)
        elif isinstance(v, list):
            cleaned[k] = [_clean_schema(i) if isinstance(i, dict) else i for i in v]
        else:
            cleaned[k] = v
    return cleaned


def _build_tool_schemas(agent_spec: dict) -> list[dict]:
    """
    Build OpenAI-compatible tool schemas from the agent spec.

    - HTTP tools: use query_params to build the schema (as before)
    - stdio tools: query the MCP server for its real tool list
    """
    mcp_tools = (agent_spec.get("tools") or {}).get("mcp") or []
    schemas = []

    for tool in mcp_tools:
        transport = tool.get("transport", {})
        transport_type = transport.get("type", "http")

        if transport_type == "stdio":
            # Discover real tools from the MCP server
            real_tools = list_stdio_tools(transport)
            for rt in real_tools:
                schemas.append({
                    "type": "function",
                    "function": {
                        "name":        rt.get("name", ""),
                        "description": rt.get("description", ""),
                        "parameters":  _clean_schema(rt.get("inputSchema", {
                            "type": "object", "properties": {}
                        })),
                    },
                    "_mcp_server": tool.get("name"),
                    "_transport":  transport,
                })
        else:
            # HTTP tool — build schema from query_params
            params: dict = {"type": "object", "properties": {}, "required": []}
            for qp in tool.get("query_params", []):
                key = qp.get("key", "")
                params["properties"][key] = {
                    "type": "string",
                    "description": qp.get("description", ""),
                }
                if qp.get("required", False):
                    params["required"].append(key)

            schemas.append({
                "type": "function",
                "function": {
                    "name":        tool.get("name", ""),
                    "description": tool.get("description", ""),
                    "parameters":  params,
                },
                "_mcp_server": tool.get("name"),
                "_transport":  transport,
            })

    return schemas


def _get_allowed_dirs(agent_spec: dict) -> list[str]:
    """Extract resolved directory paths from stdio tool args."""
    dirs = []
    mcp_tools = (agent_spec.get("tools") or {}).get("mcp") or []
    for tool in mcp_tools:
        transport = tool.get("transport", {})
        if transport.get("type") == "stdio":
            for arg in transport.get("args", []):
                resolved = _resolve_env_str(arg)
                if resolved and os.path.isdir(resolved):
                    dirs.append(resolved)
    return dirs


def _resolve_env_str(value: str) -> str:
    import re
    def replacer(match):
        return os.environ.get(match.group(1), "")
    return re.sub(r"\$\{env:([^}]+)\}", replacer, value or "")


def _build_system_prompt(agent_spec: dict) -> str | None:
    parts = []
    if agent_spec.get("role"):
        parts.append(agent_spec["role"])
    if agent_spec.get("instructions"):
        parts.append("## Instructions\n" + agent_spec["instructions"])
    if agent_spec.get("enforcement"):
        parts.append("## Enforcement\n" + agent_spec["enforcement"])
    if agent_spec.get("json_output_template"):
        parts.append(
            "## Output Format\n"
            "You MUST respond using exactly this JSON structure:\n"
            "```json\n" + agent_spec["json_output_template"] + "\n```"
        )
    else:
        parts.append("## Output Format\nRespond in a clear, friendly, conversational tone.")

    # Tell the LLM which directories it can access via filesystem tools
    allowed_dirs = _get_allowed_dirs(agent_spec)
    if allowed_dirs:
        dirs_list = "\n".join(f"- {d}" for d in allowed_dirs)
        parts.append(
            f"## Available Directories\n"
            f"When using filesystem tools, use these exact paths as the root:\n{dirs_list}\n"
            f"Always start exploration from one of these paths."
        )

    return "\n\n".join(parts) if parts else None


def chat_node(state: RuntimeState) -> RuntimeState:
    messages   = list(state.get("messages") or [])
    user_input = state.get("user_input", "")
    agent_spec = state.get("agent_spec", {})

    messages.append({"role": "user", "content": user_input})

    tool_schemas = _build_tool_schemas(agent_spec)
    system_prompt = _build_system_prompt(agent_spec)

    # Strip internal routing metadata and filter out tools with empty names
    openai_tools = [
        {k: v for k, v in s.items() if not k.startswith("_")}
        for s in tool_schemas
        if s.get("function", {}).get("name", "").strip()
    ] if tool_schemas else None

    # Store full schemas (with routing metadata) in state for tools_node
    state["_tool_schemas"] = [
        s for s in tool_schemas
        if s.get("function", {}).get("name", "").strip()
    ]

    result = generate_with_tools(
        messages=messages,
        tools=openai_tools,
        system_prompt=system_prompt,
    )

    messages.append({"role": "assistant", "content": result["response"]})

    state["messages"]           = messages
    state["assistant_response"] = result["response"]
    state["tool_calls"]         = result["tool_calls"]

    return state
