from schemas.runtime_state import RuntimeState


def router_node(state: RuntimeState):
    tool_calls = state.get("tool_calls") or []
    state["has_tool_calls"] = len(tool_calls) > 0
    return state