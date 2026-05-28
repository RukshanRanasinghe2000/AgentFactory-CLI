# AgentFactory-CLI (Runtime)

> AgentFactory CLI is a Python + Go runtime inspired by [Agent-Flavored Markdown (AFM)](https://wso2.github.io/agent-flavored-markdown/)for validating, running, and serving portable AI agents defined in .md files. It supports runtime execution powered by LangGraph and any OpenAI-compatible LLM while helping developers catch structural errors, missing fields, typos, and unsafe configurations before deployment. Special thanks to the authors at WSO2 for introducing and open-sourcing the AFM concept.

---

## Table of Contents

- [Features](#features)
- [Demos](#demos)
- [Building Agents](#building-agents)
  - [Console Chat Agent](#console-chat-agent)
  - [Code Explainer Agent](#code-explainer-agent-stdio-mcp-tools)
  - [Skills-Loaded Agent](#skills-loaded-agent)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Environment Variables](#environment-variables)
- [Tech Stack](#tech-stack)
- [Test Specs](#test-specs)
- [Project Structure](#project-structure)
- [Architecture](#architecture)

---

## Features

- **Interactive chat** — run any agent spec in your terminal with full conversation history
- **Telegram bot** — deploy agents to Telegram via polling or webhook with per-chat session memory
- **MCP tool support** — HTTP and stdio transports, dynamic tool discovery from MCP servers
- **Local skills** — inject structured markdown workflows into the agent's context  `(./skills)`
- **Output schemas** — agents with a `# Output Schema` section return structured JSON automatically
- **Multi-provider** — works with Groq, OpenAI, or any OpenAI-compatible endpoint via `.env`
- **Spec validation** — validate `.md` agent specs with the Go CLI validator before running

---

## Demos

### Console Chat Agent
https://github.com/user-attachments/assets/demo_v_1.mp4

> Weather forecast agent running in the terminal — HTTP MCP tool calling OpenWeatherMap, structured JSON output.

---

### Skills-Loaded Agent
https://github.com/user-attachments/assets/demo_v_2.mp4

> Support agent with local skills — structured troubleshooting workflows injected into the agent's context.

---

### Telegram Bot Agent
https://github.com/user-attachments/assets/demo_v_3.mp4

> Telegram bot running in polling mode — per-chat session memory, live responses via the Telegram API.

---

## Building Agents

### Console Chat Agent

A conversational agent that runs in your terminal and calls HTTP APIs as tools.

**1. Create your spec file** (e.g. `my-agent.md`):



**2. Add your API keys to `.env`:**

```env
MODEL_API_KEY=your-groq-key
OPENWEATHER_API_KEY=your-openweathermap-key
```

**3. Run it:**

```bash
.\agentfactory.exe run my-agent.md
```

---

### Code Explainer Agent (stdio MCP tools)

An agent that reads and explains a real codebase using the filesystem MCP server.

**Prerequisites:** Node.js must be installed (`npx` available on PATH).

**1. Set `PROJECT_DIR` in `.env`** to the directory you want the agent to explore:

```env
MODEL_API_KEY=your-groq-key
PROJECT_DIR=C:\path\to\your\project
```

**2. Create your spec file (code-explainer.md)**


**3. Run it:**

```bash
.\agentfactory.exe run code-explainer.md
```

Then ask: `What does this project do?` or `Explain the main entry point.`

---

### Skills-Loaded Agent

An agent that follows structured troubleshooting workflows defined in local markdown skill files.

**1. Create a `skills/` folder** next to your spec file with one `.md` file per skill:

```
my-agent/
  agent.md
  skills/
    account-troubleshoot.md
    connectivity-troubleshoot.md
```

**2. Write a skill file** (`skills/account-troubleshoot.md`):

**3. Create your spec file** (`agent.md`):

**4. Run it:**

```bash
.\agentfactory.exe run my-agent/agent.md
```

Then ask: `I can't log into my account` — the agent will follow the account troubleshooting skill workflow.

---

## Quick Start

```bash
cd af_application
uv venv && uv sync

# 2. Copy and fill in your API keys
cp .env.example .env

# 3. Build the Go CLI
cd cmd && go build -o ../agentfactory.exe . && cd ..

# 4. Run an agent
.\agentfactory.exe run test_data/weather.md
```

---

## Commands

| Command | Description |
|---|---|
| `agentfactory run <agent.md>` | Interactive chat in the terminal |
| `agentfactory serve <agent.md>` | Start Telegram bot (polling or webhook) |
| `agentfactory version` | Show version |

---

## Environment Variables

```env
MODEL_PROVIDER=groq
MODEL_NAME=llama-3.3-70b-versatile
MODEL_BASE_URL=https://api.groq.com/openai/v1
MODEL_API_KEY=your-groq-key

OPENWEATHER_API_KEY=your-key     # for weather agent
TELEGRAM_BOT_TOKEN=your-token    # for telegram agent
PROJECT_DIR=/path/to/project     # for code explainer agent
```

---

## Tech Stack

| Technology | Purpose |
|---|---|
| **Go** | CLI binary — `run`, `serve`, `vali` commands, HTTP server for Telegram |
| **Python 3.13** | AI workflow runtime — LangGraph orchestration, LLM calls, tool execution |
| **LangGraph** | Stateful agent graph — `chat → router → tools → response` workflow |
| **LangChain** | LLM abstractions and prompt utilities |
| **OpenAI SDK** | Provider client — works with Groq, OpenAI, and any OpenAI-compatible endpoint |
| **Pydantic** | Schema validation — `AgentSpec`, `RuntimeState`, all data models |
| **PyYAML** | Parses YAML frontmatter from `.md` spec files |
| **python-dotenv** | Loads `.env` API keys into the Python runtime |
| **requests** | HTTP transport for MCP tool calls |
| **uv** | Fast Python package manager and virtual environment |

---

## Test Specs

| File | What it tests |
|---|---|
| `test_data/weather.md` | HTTP MCP tool — OpenWeatherMap |
| `test_data/math_tutor.md` | Conversational, no tools |
| `test_data/code_explainer.md` | stdio MCP — filesystem + sequential-thinking |
| `test_data/support_agent.md` | Local skills — account and connectivity troubleshooting |
| `test_data/telegram_agent.md` | Telegram platformchat — polling mode |

```bash
python test_runtime.py test_data/weather.md "Weather in Colombo?"
python test_runtime.py test_data/math_tutor.md "Derivative of x squared?"
```

---

## Project Structure

```
af_application/
  cmd/              Go CLI (run, serve, init)
  graphs/           LangGraph workflow graphs
  nodes/            chat, router, tools, response nodes
  adapters/         spec_loader, runtime_bridge, skills_adapter, runtime_adapter
  providers/        openai.py — Groq/OpenAI compatible provider
  schemas/          Pydantic models — AgentSpec, RuntimeState
  prompts/          Markdown prompt templates
  test_data/        Sample agent specs and skills
  docs/             Architecture diagrams 
```

---

## Architecture

See [`docs/architecture.md`](docs/architecture.md) for Mermaid diagrams covering the full system, runtime graph, tool execution, platform chat, and memory architecture.
