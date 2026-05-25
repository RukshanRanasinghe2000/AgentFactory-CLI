from pydantic import BaseModel
from typing import Dict, List, Optional


class Authentication(BaseModel):
    type: str
    api_key: Optional[str] = None       # api-key auth
    username: Optional[str] = None      # basic auth
    password: Optional[str] = None      # basic auth

    model_config = {"extra": "allow"}


class Transport(BaseModel):
    type: str                            # "http" | "stdio"
    url: Optional[str] = None           # http transport
    command: Optional[str] = None       # stdio transport
    args: List[str] = []
    env: Optional[Dict[str, str]] = None

    model_config = {"extra": "allow"}


class QueryParam(BaseModel):
    key: str
    description: str = ""
    required: bool = False


class MCPTool(BaseModel):
    name: str
    transport: Transport
    authentication: Optional[Authentication] = None   # not required for stdio
    query_params: List[QueryParam] = []

    model_config = {"extra": "allow"}


class Tool(BaseModel):
    name: str
    description: Optional[str] = None


class Tools(BaseModel):
    mcp: List[MCPTool] = []
    builtin: List[Tool] = []

    model_config = {"extra": "allow"}


class Interface(BaseModel):
    type: str

    model_config = {"extra": "allow"}


class ModelConfig(BaseModel):
    provider: str
    name: str
    temperature: float = 0.7
    authentication: Optional[Authentication] = None

    model_config = {"extra": "allow"}


class AgentSpec(BaseModel):
    spec_version: Optional[str] = None
    name: str
    description: str
    version: str
    license: Optional[str] = None
    author: Optional[str] = None

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

    model_config = {"extra": "ignore"}
