from schemas.runtime_state import RuntimeState


def router_node(state: RuntimeState):
    """
    Decide whether tools should execute.
    """

    tool_calls = state.get("tool_calls", [])

    state["has_tool_calls"] = len(tool_calls) > 0

    return state