from schemas.agent_spec import AgentSpec


def validate_spec(spec_data: dict):
    """
    Temporary validation adapter.

    Later this can call the Go validator.
    """

    try:
        AgentSpec(**spec_data)

        return {
            "valid": True,
            "errors": []
        }

    except Exception as e:
        return {
            "valid": False,
            "errors": [str(e)]
        }