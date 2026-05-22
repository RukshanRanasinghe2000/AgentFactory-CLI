from pydantic import ValidationError

from schemas.state import AgentState
from schemas.agent_spec import AgentSpec


def validate_node(state: AgentState):
    """
    Validate generated AgentSpec.
    """

    spec_data = state.get("agent_spec")

    if not spec_data:
        state["validation_errors"] = [
            "No agent spec found"
        ]

        return state

    try:
        validated_spec = AgentSpec(**spec_data)

        state["agent_spec"] = validated_spec.model_dump()
        state["validation_errors"] = []

    except ValidationError as e:
        state["validation_errors"] = [
            err["msg"]
            for err in e.errors()
        ]

    return state