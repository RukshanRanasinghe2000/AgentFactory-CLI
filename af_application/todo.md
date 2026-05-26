# TODO.md

# AgentFactory AI + LangGraph Roadmap

---

# Phase 1 — Foundation Setup

## Environment

- [x] Create `ai/` workspace
- [x] Setup Python virtual environment using `uv`
- [x] Initialize `pyproject.toml`
- [x] Install LangGraph
- [x] Install LangChain
- [x] Install OpenAI SDK
- [x] Install Pydantic
- [x] Install PyYAML
- [x] Install python-dotenv
- [x] Install requests

---

# Phase 2 — Project Structure

## Core Directories

- [x] Create `graphs/`
- [x] Create `nodes/`
- [x] Create `prompts/`
- [x] Create `schemas/`
- [x] Create `providers/`
- [x] Create `adapters/`
- [x] Create `exports/`

---

# Phase 3 — Schema System

## Core Schemas

- [x] Create `schemas/agent_spec.py`
- [x] Create `schemas/state.py`
- [x] Create `schemas/validation.py`

## AgentSpec Models

- [x] Authentication schema
- [x] ModelConfig schema
- [x] Interface schema
- [x] QueryParam schema
- [x] Transport schema
- [x] MCPTool schema
- [x] Tools schema
- [x] AgentSpec schema
- [x] AgentState schema
- [x] ValidationResult schema

---

# Phase 4 — Prompt System

## Prompt Infrastructure

- [x] Create `prompts/loader.py`
- [x] Create `clarify_agent.md`
- [x] Create `refine_agent.md`

## Prompt Features

- [x] Markdown-based prompts
- [x] Prompt loader system
- [x] Reusable prompt architecture

---

# Phase 5 — Provider Layer

## OpenAI Provider

- [x] Create `providers/openai.py`
- [x] Implement text generation
- [x] Implement basic provider abstraction

---

# Phase 6 — Workflow Nodes

## Core Nodes

- [x] Create `clarify.py`
- [x] Create `generate.py`
- [x] Create `validate.py`
- [x] Create `export.py`

## Node Responsibilities

- [x] Clarification question generation
- [x] AgentSpec generation
- [x] Pydantic validation
- [x] `.md` export generation

---

# Phase 7 — LangGraph Workflow

## Initial Graph

- [x] Create `graphs/init_graph.py`
- [x] Register nodes
- [x] Define graph edges
- [x] Compile workflow

## Current Workflow

- [x] START → clarify
- [x] clarify → generate
- [x] generate → validate
- [x] validate → export
- [x] export → END

---

# Phase 8 — Adapters

## Core Adapters

- [x] Create `go_bridge.py`
- [x] Create `validator_adapter.py`
- [x] Create `runtime_adapter.py`

## Adapter Features

- [x] Go ↔ Python communication
- [x] JSON workflow bridge
- [x] Validation adapter structure
- [x] Runtime adapter structure

---

# Phase 9 — Immediate MVP Tasks

## Prompt Cleanup

- [ ] Move uploaded runtime prompts into:
  - [ ] `clarify_agent.md`
  - [ ] `refine_agent.md`
- [ ] Remove architecture notes
- [ ] Remove implementation docs
- [ ] Keep only LLM instructions
- [ ] Add output formatting rules

---

#  Provider Improvements

## OpenAI Provider

- [ ] Add `.env` support
- [ ] Load API keys
- [ ] Add error handling
- [ ] Add retries
- [ ] Add timeout handling
- [ ] Add structured JSON outputs

---

#  Workflow Testing

## Graph Testing

- [ ] Run LangGraph workflow
- [ ] Verify state transitions
- [ ] Verify prompt loading
- [ ] Verify provider execution
- [ ] Verify JSON parsing
- [ ] Verify validation
- [ ] Verify export generation
- [ ] Verify error handling

---

#  Export System

## AFM Export Improvements

- [ ] Improve YAML formatting
- [ ] Add configurable exports
- [ ] Add overwrite protection
- [ ] Add prettier markdown output
- [ ] Add export cleanup

---

#  Go Integration

## CLI Integration

- [ ] Execute Python workflow from Go
- [ ] Send stdin JSON
- [ ] Receive stdout JSON
- [ ] Parse workflow output
- [ ] Handle Python errors
- [ ] Add CLI integration
- [ ] Add `agentfactory init`

---

#  Phase 10 — Additional Graphs

## Advanced Graphs

- [ ] Create `clarify_graph.py`
- [ ] Create `refine_graph.py`
- [ ] Create `fix_graph.py`
- [ ] Create `runtime_graph.py`
- [ ] Create `tool_graph.py`

---

#  Phase 11 — Additional Nodes

## Advanced Nodes

- [ ] Create `repair.py`
- [ ] Create `tools.py`
- [ ] Create runtime execution nodes
- [ ] Create retry nodes
- [ ] Create loop nodes
- [ ] Create planning nodes
- [ ] Create tool-selection nodes

---

#  Phase 12 — Multi-Provider System

## Providers

- [ ] Create `groq.py`
- [ ] Create `ollama.py`
- [ ] Add Anthropic provider
- [ ] Add provider router
- [ ] Add provider fallback system
- [ ] Add provider configuration layer

## Multi-Provider Abstractions

- [ ] Unified provider interface
- [ ] Dynamic provider selection
- [ ] Model capability registry
- [ ] Provider failover logic
- [ ] Provider-specific prompt injection

---

#  Phase 13 — Runtime System

## Runtime Execution

- [ ] MCP runtime execution
- [ ] Authentication injection
- [ ] Query param injection
- [ ] HTTP transport support
- [ ] SSE transport support
- [ ] WebSocket transport support

## Runtime Features

- [ ] Runtime sessions
- [ ] Streaming responses
- [ ] Runtime observability
- [ ] Runtime logging
- [ ] Runtime metrics

---

#  Phase 14 — Tool System

## Tool Runtime

- [ ] Dynamic tool execution
- [ ] Tool registries
- [ ] Tool discovery
- [ ] Tool validation
- [ ] Tool schema parsing

## MCP Features

- [ ] MCP tool registry
- [ ] MCP authentication system
- [ ] MCP transport abstraction
- [ ] MCP capability negotiation

---

#  Phase 15 — Validation Engine

## Validation Improvements

- [ ] Connect Go validator
- [ ] Position-aware diagnostics
- [ ] Rule-based validation
- [ ] Typo detection integration
- [ ] Validation pipelines

## Auto-Repair

- [ ] AI repair workflow
- [ ] Validation-aware fixing
- [ ] Auto-fix suggestions
- [ ] Spec correction loops

---

#  Phase 16 — Memory Systems

## Memory Architecture

- [ ] Short-term memory
- [ ] Long-term memory
- [ ] Session memory
- [ ] Context compression
- [ ] Vector memory store

## Memory Features

- [ ] Conversation persistence
- [ ] Tool usage memory
- [ ] Runtime memory
- [ ] Agent learning memory

---

#  Phase 17 — Agent Loops

## Agentic Runtime

- [ ] Iterative agent loops
- [ ] Self-reflection loops
- [ ] Planning loops
- [ ] Retry loops
- [ ] Tool reasoning loops

## Execution Control

- [ ] Max iteration limits
- [ ] Stop conditions
- [ ] Goal evaluation
- [ ] Loop monitoring

---

#  Phase 18 — Web Builder

## UI Builder

- [ ] Landing page
- [ ] Idea input UI
- [ ] Clarification UI
- [ ] Spec builder UI
- [ ] Live preview UI

## Interactive Features

- [ ] AI-assisted generation
- [ ] Real-time validation
- [ ] Live YAML preview
- [ ] Interactive testing
- [ ] Runtime playground

---

#  Phase 19 — Runtime Playground

## Testing Environment

- [ ] Live chat runtime
- [ ] Tool execution testing
- [ ] Runtime debugging
- [ ] Agent monitoring
- [ ] Session replay

---

#  Phase 20 — Deployment System

## Deployment

- [ ] AFM packaging
- [ ] Runtime deployment
- [ ] Spec publishing
- [ ] Runtime hosting
- [ ] Cloud deployment

---

#  MVP Goal

## Target Command

- [ ] `agentfactory init`

## MVP Workflow

- [ ] User enters idea
- [ ] AI asks clarification questions
- [ ] AI generates AgentSpec
- [ ] Validator validates spec
- [ ] Export `agent.md`

---

#  Architecture Validation Goal

- [ ] Validate Go ↔ Python bridge
- [ ] Validate LangGraph orchestration
- [ ] Validate AFM generation
- [ ] Validate export pipeline
- [ ] Complete end-to-end workflow