from typing import TypedDict, List, Optional
from schemas.agent_spec import AgentSpec

class AgentState(TypedDict):
    idea: str
    clarification_questions: List[str]
    clarification_answers: List[str]
    enriched_idea: str
    agent_spec: Optional[AgentSpec]
    validation_errors: List[str]