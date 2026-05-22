import json

from schemas.state import AgentState
from prompts.loader import load_prompt
from providers.openai import generate_text


def generate_node(state: AgentState):
    """
    Generate AgentSpec JSON from idea and clarification answers.
    """

    idea = state["idea"]
    answers = state.get("clarification_answers", [])

    prompt_template = load_prompt("refine_agent.md")

    formatted_answers = "\n".join([
        f"- {answer}"
        for answer in answers
    ])

    prompt = prompt_template.format(
        idea=idea,
        answers=formatted_answers
    )

    response = generate_text(prompt)

    try:
        spec = json.loads(response)
    except json.JSONDecodeError:
        state["validation_errors"] = [
            "Failed to parse generated AgentSpec JSON"
        ]
        return state

    state["agent_spec"] = spec
    state["validation_errors"] = []

    return state