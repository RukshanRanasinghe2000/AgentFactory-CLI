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
    tool_results = state.get("tool_results", [])
    messages     = list(state.get("messages", []))
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

    # Ask the LLM to synthesise a final answer from the tool results
    system_prompt = agent_spec.get("role") or agent_spec.get("instructions") or None

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
