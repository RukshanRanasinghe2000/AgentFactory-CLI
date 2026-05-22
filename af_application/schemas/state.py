from typing import TypedDict, List, Optional, Any

class AgentState(TypedDict):
    idea: str
    clarification_questions: List[str]
    clarification_answers: List[str]
    enriched_idea: str
    agent_spec: Optional[Any]  # dict during generation, validated dict after validate_node
    validation_errors: List[str]
    exported_file: Optional[str]