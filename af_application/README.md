# AgentFactory-CLI (Runtime)

> Python + Go runtime for running `.md` agent specs powered by LangGraph and any OpenAI-compatible LLM.

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

## Quick Start

```bash
# 1. Set up Python environment
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
