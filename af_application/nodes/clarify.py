from schemas.state import AgentState
from prompts.loader import load_prompt
from providers.openai import generate_text


def clarify_node(state: AgentState):
    """
    Generate clarification questions based on the user's idea.
    """

    idea = state["idea"]

    prompt_template = load_prompt("clarify_agent.md")

    prompt = prompt_template.format(
        idea=idea
    )

    response = generate_text(prompt)

    questions = [
        q.strip("- ").strip()
        for q in response.splitlines()
        if q.strip()
    ]

    state["clarification_questions"] = questions

    return state