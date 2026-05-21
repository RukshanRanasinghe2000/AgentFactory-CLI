from pydantic import BaseModel
from typing import List, Optional

class Authentication(BaseModel):
    type: str
    api_key: str
     
class ModelConfig(BaseModel):
    provider: str
    name: str
    temperature: float = 0.7
    authentication: Authentication

class Interface(BaseModel):
    type: str

class QueryParam(BaseModel):
    key: str
    description: str
    required: bool = False

class Transport(BaseModel):
    type: str
    url: str

class MCPTool(BaseModel):
    name: str
    transport: Transport
    authentication: Authentication
    query_params: List[QueryParam] = []

class Tool(BaseModel):
    name: str
    description: Optional[str] = None

class AgentSpec(BaseModel):
    spec_version: Optional[str] = None
    name: str
    description: str
    version: str
    license: str

    max_iterations: int = 5
    execution_mode: str = "agentic"

    model: ModelConfig

    interfaces: List[Interface] = []

    tools: Tools