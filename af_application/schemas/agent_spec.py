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

class Tools(BaseModel):
    mcp: List[MCPTool] = []
    builtin: List[Tool] = []

class AgentSpec(BaseModel):
    spec_version: Optional[str] = None
    name: str
    description: str
    version: str
    license: Optional[str] = None

    role: Optional[str] = None
    instructions: Optional[str] = None
    output_format: Optional[str] = None
    enforcement: Optional[str] = None
    memory_type: Optional[str] = None
    json_output_template: Optional[str] = None

    max_iterations: int = 5
    execution_mode: str = "sequential"

    model: Optional[ModelConfig] = None
    interfaces: List[Interface] = []
    tools: Optional[Tools] = None

    suggested_tools: List[str] = []
    suggested_interfaces: List[str] = []