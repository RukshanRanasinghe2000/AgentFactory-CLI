from schemas.runtime_state import RuntimeState
from adapters.runtime_adapter import execute_tool_call


def find_tool(agent_spec, tool_name):
    """
    Find tool definition from agent spec.
    """

    mcp_tools = (
        agent_spec
        .get("tools", {})
        .get("mcp", [])
    )

    for tool in mcp_tools:
        if tool.get("name") == tool_name:
            return tool

    return None


def build_tool_url(tool_definition, arguments):
    """
    Replace URL placeholders using arguments.
    """

    url = (
        tool_definition
        .get("transport", {})
        .get("url", "")
    )

    for key, value in arguments.items():
        placeholder = f"{{{key}}}"
        url = url.replace(placeholder, str(value))

    return url


def tools_node(state: RuntimeState):
    """
    Execute dynamically requested tools.
    """

    agent_spec = state.get("agent_spec", {})
    tool_calls = state.get("tool_calls", [])

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
            "tool": tool_name,
            "success": result.get("success", False),
            "result": result
        })

    state["tool_results"] = tool_results

    return state