# AgentFactory-CLI Runtime ( Abstract Architecture )

## 1. Top-Level: User to CLI to Python

> The Go binary is the single entry point. It routes `vali`, `run`, and `serve` commands to the appropriate subsystem — the Go validator for spec checking, or the Python runtime bridge for agent execution.

```mermaid
flowchart LR

    %% User
    U([User])

    %% CLI Layer
    subgraph CLI["Go CLI"]
        VALI["vali command"]
        RUN["run command"]
        SERVE["serve command"]
    end

    %% Validation Flow
    subgraph VALIDATION["Validation Pipeline"]
        PARSER["lexer + parser + validator"]
    end

    %% Runtime Flow
    subgraph RUNTIME["Python Runtime"]
        RB["runtime_bridge.py"]
        SL["spec_loader.py"]
        RG["runtime_graph"]
    end

    %% Server Flow
    subgraph SERVER["Serving Layer"]
        HTTP["Go HTTP Server"]
        TG["Telegram API"]
    end

    %% User Commands
    U -->|"agentfactory vali agent.md"| VALI
    U -->|"agentfactory run agent.md"| RUN
    U -->|"agentfactory serve agent.md"| SERVE

    %% Validation
    VALI -->|"parse + validate"| PARSER
    PARSER -->|"validation report"| VALI
    VALI -->|"results"| U

    %% Runtime
    RUN -->|"load/chat"| RB
    RB -->|"load spec"| SL
    SL -->|"AgentSpec"| RB
    RB -->|"invoke"| RG
    RG -->|"assistant response"| RB
    RB -->|"JSON response"| RUN
    RUN -->|"response"| U

    %% Serve Mode
    SERVE -->|"load"| RB
    SERVE -->|"start server"| HTTP
    HTTP -->|"webhook events"| RB
    RB -->|"assistant response"| HTTP

    %% Telegram
    HTTP -->|"sendMessage"| TG
    TG -->|"updates"| HTTP
```

---

## 2. Runtime Graph: LLM and Tools

> The LangGraph `StateGraph` orchestrates each chat turn. The LLM decides whether to call tools or respond directly. Tool results are fed back to the LLM for a grounded final answer before the turn ends.

```mermaid
graph TD
    RG[runtime_graph] --> CN[chat_node]
    CN -->|messages + tool schemas| LLM[LLM API]
    LLM -->|response + tool_calls| CN
    CN --> RN[router_node]

    RN -->|tool_calls present| TN[tools_node]
    RN -->|no tool_calls| RSN[response_node]
    TN --> RSN

    TN -->|HTTP transport| HTTP[execute_http_tool]
    TN -->|stdio transport| STDIO[execute_stdio_tool]

    HTTP --> API[HTTP API]
    STDIO --> MCP[stdio MCP Server]

    RSN -->|tool results + messages| LLM
    LLM -->|final answer| RSN
    RSN -->|finished=true| RG
```

---

## 3. Spec Loading and Skills

> The spec loader parses the `.md` file into a typed `AgentSpec` dict. Skills are loaded from local markdown files and injected into the LLM system prompt — no code required to add new workflows.

```mermaid
graph TD
    AFM[(agent.md)] --> SL[spec_loader.py]
    SL -->|YAML frontmatter| SPEC[AgentSpec dict]
    SL -->|Role section| SPEC
    SL -->|Instructions section| SPEC
    SL -->|Enforcement section| SPEC
    SL -->|Output Schema section| SPEC

    SPEC --> CN[chat_node]
    CN --> SA[skills_adapter.py]
    SA --> SK[(skills folder)]
    SK -->|skill md files| SA
    SA -->|skills context| CN
    CN -->|system prompt| LLM[LLM API]
```

---

## 4. Platform Chat: Telegram

> The `serve` command reads the spec's `platformchat` interface, resolves the bot token from `.env`, and starts either a polling loop or an HTTP webhook handler. Each Telegram chat gets its own isolated session history.

```mermaid
graph TD
    SPEC[(telegram_agent.md)] -->|interfaces type=platformchat| SERVE[agentfactory serve]
    SERVE -->|reads .env| ENV[TELEGRAM_BOT_TOKEN]
    ENV --> SERVE

    SERVE -->|mode=polling| POLL[Telegram getUpdates loop]
    SERVE -->|mode=notification| WH[HTTP webhook handler]

    POLL -->|new message text| BRIDGE[runtime_bridge.py webhook phase]
    WH -->|POST body text| BRIDGE

    BRIDGE -->|session_id = chat_id| SESSION[per-chat message history]
    SESSION --> RG[runtime_graph]
    RG -->|assistant_response| BRIDGE
    BRIDGE -->|response text| POLL
    BRIDGE -->|response text| WH

    POLL -->|sendMessage| TG[Telegram API]
    WH -->|sendMessage| TG
    TG -->|delivered| USER([Telegram User])
```

---

## 5. Validation Flow

> Before running, the Go CLI calls the `af_cli` validator binary against the spec file. The validator runs the lexer, parser, and rule engine — checking typos, required fields, auth patterns, and section content — then prints a colored report.

```mermaid
graph TD
    U([User]) -->|agentfactory vali agent.md| MAIN[main.go]
    MAIN --> VC[cmd/validate.go]
    VC --> CFG[config.go - load .afvalidate.toml]
    VC --> LEX[lexer.go - tokenize]
    LEX --> PAR[parser.go - parse spec]
    PAR --> VAL[validator.go - run rules]
    VAL -->|errors| VC
    VAL -->|warnings| VC
    VAL -->|passed| VC
    VC -->|colored report| U
```

---

## 6. Tool Execution: HTTP vs stdio

> Tools are routed by transport type. HTTP tools make a direct API call. stdio tools spawn an MCP server process (e.g. `npx @modelcontextprotocol/server-filesystem`), perform the JSON-RPC handshake, and call the tool — all transparently.

```mermaid
graph TD
    TC[Tool Call from LLM] --> ROUTE{transport type}

    ROUTE -->|http| HTTP[execute_http_tool]
    ROUTE -->|stdio| STDIO[execute_stdio_tool]

    HTTP --> API[HTTP API]
    API -->|JSON response| HTTP

    STDIO --> PROC[spawn process via npx or uvx]
    PROC -->|JSON-RPC initialize| PROC
    PROC -->|JSON-RPC tools/call| PROC
    PROC -->|JSON-RPC response| STDIO

    HTTP --> RESULT[tool_result]
    STDIO --> RESULT
    RESULT --> RSN[response_node]
```

---

## 7. Memory Architecture

> `memory_type` is set in the YAML frontmatter of the agent spec. `platformchat` session isolation is not a `memory_type` — it activates automatically when `interfaces: [{type: "platformchat"}]` is set, switching the runtime to per-chat session handling.

```yaml
memory_type: "short-term"   # none | short-term | long-term | semantic
```

| Goal | Config |
|---|---|
| Stateless, no history | `memory_type: "none"` |
| Conversational terminal agent | `memory_type: "short-term"` (default) |
| Telegram bot, per-user memory | `interfaces: [{type: "platformchat"}]` |
| Persist memory across restarts | `memory_type: "long-term"` *(planned)* |
| Vector-based recall | `memory_type: "semantic"` *(planned)* |

**Why LangGraph?** `RuntimeState` is a typed dict passed through every node — memory is just a field in that dict. Conditional edges handle looping and exit. Checkpointers (SQLite, Redis) can persist full graph state with zero changes to node logic.

**`none`** — No history passed to the LLM. Every turn is a fresh context. Good for lookup tools, formatters, or one-shot agents where prior messages add noise.

**`short-term`** *(default)* — Last 10 messages kept in `RuntimeState.messages`, passed to the LLM each turn. Tool results truncated to 2000 chars to avoid context bloat. One session per process — all messages in a `run` session belong to the same conversation. History is lost on exit.

**`platformchat`** — Short-term memory extended for multi-user deployments. Sessions keyed by `platform + chat_id` in an in-process dict in `runtime_bridge.py`. Each Telegram chat gets isolated history — concurrent users never mix. Lost on process restart.

**`long-term`** *(planned)* — LangGraph checkpointer persists full graph state to SQLite or Redis. Sessions keyed by `thread_id` — the runtime reloads state on the next turn with the same ID. A summarizer compresses old turns to keep the context window manageable.

**`semantic`** *(planned)* — User messages are embedded and stored in a vector DB (Pinecone, pgvector), namespaced by `user_id`. Each turn retrieves the top-K most relevant past memories and injects them into the system prompt. Best for agents with very long histories where replaying raw messages would overflow context.

```mermaid
flowchart TD

    MT["memory_type"]

    %% none
    subgraph NONE["none"]
        N1["No history passed"]
        N2["Each turn independent"]
        N1 --> N2
    end

    %% short-term
    subgraph ST["short-term"]
        S1["Rolling window"]
        S2["Tool results truncated"]
        S3["Messages in RuntimeState"]
        S4["Passed to LLM each turn"]

        S1 --> S3
        S2 --> S3
        S3 --> S4
    end

    %% platformchat
    subgraph PLAT["platformchat"]
        P1["session = platform + chat_id"]
        P2["In-process session store"]
        P3["Per-chat isolated history"]

        P1 --> P2
        P2 --> P3
    end

    %% long-term
    subgraph LT["long-term"]
        L1["SQLite / Redis checkpointer"]
        L2["Load graph state"]
        L3["Auto-save state"]
        L4["Summarize old turns"]

        L3 --> L1
        L1 --> L2
        L4 --> L2
    end

    %% semantic
    subgraph SEM["semantic"]
        V1["Embed messages"]
        V2["Vector DB"]
        V3["Retrieve memories"]
        V4["Inject into prompt"]

        V1 --> V2
        V2 --> V3
        V3 --> V4
    end

    %% llm input
    subgraph INPUT["LLM Input"]
        I1["System Prompt"]
        I2["Message History"]
        I3["Tool Results"]
    end

    MT -->|none| NONE
    MT -->|short-term| ST
    MT -->|platformchat| PLAT
    MT -->|long-term| LT
    MT -->|semantic| SEM

    S4 --> I2
    P3 --> I2
    L2 --> I2
    V4 --> I1

```
