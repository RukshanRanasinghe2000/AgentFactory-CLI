from schemas.runtime_state import RuntimeState


def response_node(state: RuntimeState):
    """
    Finalize assistant response.
    """

    assistant_response = state.get(
        "assistant_response",
        ""
    )

    tool_results = state.get(
        "tool_results",
        []
    )

    if tool_results:

        formatted_results = []

        for result in tool_results:

            tool_name = result.get("tool")

            if result.get("success"):

                formatted_results.append(
                    f"{tool_name}: "
                    f"{result['result'].get('data', '')}"
                )

            else:

                formatted_results.append(
                    f"{tool_name}: "
                    f"{result.get('error', 'Unknown error')}"
                )

        tool_output = "\n".join(formatted_results)

        assistant_response = (
            f"{assistant_response}\n\n"
            f"Tool Results:\n"
            f"{tool_output}"
        )

    state["assistant_response"] = assistant_response

    return state