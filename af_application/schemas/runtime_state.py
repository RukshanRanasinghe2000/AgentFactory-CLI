from typing import TypedDict, List, Dict, Any, Optional


class RuntimeState(TypedDict):
    """
    Shared state used by the runtime graph.
    """

    # Parsed AFM spec
    agent_spec: Dict[str, Any]

    # User message
    user_input: str

    # Chat history
    messages: List[Dict[str, str]]

    # Current LLM response
    assistant_response: Optional[str]

    # Tool calls requested by model
    tool_calls: List[Dict[str, Any]]

    # Tool execution results
    tool_results: List[Dict[str, Any]]

    # Runtime errors
    errors: List[str]

    # Session status
    finished: bool