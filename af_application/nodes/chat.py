from schemas.runtime_state import RuntimeState
from providers.openai import generate_with_tools


def _extract_tools(agent_spec: dict) -> list[dict]:
    """Pull MCP tool definitions out of the agent spec."""
    tools = agent_spec.get("tools") or {}
    return tools.get("mcp") or []


def chat_node(state: RuntimeState) -> RuntimeState:
    """
    Send the conversation + available tools to the LLM.
    Receives either a plain response or structured tool calls.
    """
    messages    = list(state.get("messages") or [])
    user_input  = state.get("user_input", "")
    agent_spec  = state.get("agent_spec", {})

    # Append the latest user turn
    messages.append({"role": "user", "content": user_input})

    # Build system prompt: role + instructions + enforcement + output schema
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
        parts.append(
            "## Output Format\n"
            "Respond in a clear, friendly, conversational tone."
        )
    system_prompt = "\n\n".join(parts) if parts else None

    # Collect tool schemas from the spec
    tools = _extract_tools(agent_spec)

    # Call LLM — structured output with optional tool calls
    result = generate_with_tools(
        messages=messages,
        tools=tools if tools else None,
        system_prompt=system_prompt,
    )

    assistant_response = result["response"]
    tool_calls         = result["tool_calls"]

    # Record assistant turn in history (content may be empty when tool_calls present)
    messages.append({"role": "assistant", "content": assistant_response})

    state["messages"]            = messages
    state["assistant_response"]  = assistant_response
    state["tool_calls"]          = tool_calls

    return state
