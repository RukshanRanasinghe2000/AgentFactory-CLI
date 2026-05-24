from schemas.runtime_state import RuntimeState
from providers.openai import generate_text


def build_prompt(messages):
    """
    Convert chat messages into a prompt string.
    """

    return "\n".join([
        f"{m['role']}: {m['content']}"
        for m in messages
    ])


def chat_node(state: RuntimeState):
    """
    Send conversation to the LLM
    and receive structured output.
    """

    messages = state.get("messages", [])
    user_input = state.get("user_input", "")

    # Append latest user message
    messages.append({
        "role": "user",
        "content": user_input
    })

    prompt = build_prompt(messages)

    result = generate_text(prompt)

    assistant_response = result.get("response", "")
    tool_calls = result.get("tool_calls", [])

    # Save assistant response
    messages.append({
        "role": "assistant",
        "content": assistant_response
    })

    state["messages"] = messages
    state["assistant_response"] = assistant_response
    state["tool_calls"] = tool_calls

    return state