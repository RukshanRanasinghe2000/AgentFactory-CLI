import json
from schemas.runtime_state import RuntimeState
from providers.openai import generate_with_tools


def response_node(state: RuntimeState) -> RuntimeState:
    """
    Finalise the assistant response.

    If tool results exist, send them back to the LLM so it can
    produce a grounded natural-language answer.
    Otherwise the existing assistant_response is used as-is.
    """
    tool_results = state.get("tool_results") or []
    messages     = list(state.get("messages") or [])
    agent_spec   = state.get("agent_spec", {})

    if not tool_results:
        # No tools were called — response is already in state
        return state

    # Inject tool results as tool-role messages so the LLM can reason over them
    for result in tool_results:
        tool_id   = result.get("id", result.get("tool", "unknown"))
        tool_name = result.get("tool", "unknown")

        if result.get("success"):
            content = json.dumps(result.get("result", {}))
        else:
            content = json.dumps({"error": result.get("error", "Tool failed")})

        messages.append({
            "role":         "tool",
            "tool_call_id": tool_id,
            "name":         tool_name,
            "content":      content,
        })

    # Build system prompt consistently with chat_node
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

    final = generate_with_tools(
        messages=messages,
        tools=None,          # no more tool calls in this turn
        system_prompt=system_prompt,
    )

    final_response = final["response"]

    # Append final assistant turn to history
    messages.append({"role": "assistant", "content": final_response})

    state["messages"]           = messages
    state["assistant_response"] = final_response

    return state
