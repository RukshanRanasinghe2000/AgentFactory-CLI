from schemas.runtime_state import RuntimeState
from providers.openai import generate_with_tools


def _extract_tools(agent_spec: dict) -> list[dict]:
    """Pull MCP tool definitions out of the agent spec."""
    return agent_spec.get("tools", {}).get("mcp", [])


def chat_node(state: RuntimeState) -> RuntimeState:
    """
    Send the conversation + available tools to the LLM.
    Receives either a plain response or structured tool calls.
    """
    messages    = list(state.get("messages", []))
    user_input  = state.get("user_input", "")
    agent_spec  = state.get("agent_spec", {})

    # Append the latest user turn
    messages.append({"role": "user", "content": user_input})

    # Build system prompt from spec role field
    system_prompt = agent_spec.get("role") or agent_spec.get("instructions") or None

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
