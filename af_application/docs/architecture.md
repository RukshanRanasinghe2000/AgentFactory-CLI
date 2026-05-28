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

> Memory is declared in the spec via `memory_type`. The runtime implements the active strategy. LangGraph's `StateGraph` is the foundation — state flows through every node as an immutable snapshot, making memory management explicit, testable, and easy to extend.

**Why LangGraph for memory and looping?**

- **Explicit state** — `RuntimeState` is a typed dict passed through every node. Memory is just a field in that dict — no hidden globals, no framework magic.
- **Conditional edges** — the `should_continue` edge lets the graph loop back to `chat_node` for multi-turn agentic workflows, or exit cleanly when `finished=True`.
- **Checkpointing** — LangGraph supports persistent checkpointers (SQLite, Redis) that can save and restore full graph state between turns, enabling long-term memory with zero changes to node logic.
- **Composable** — short-term, long-term, and semantic memory are just different implementations of the same `messages` field. Swapping strategies doesn't change the graph structure.

```mermaid
graph TD
    subgraph DECL["Spec Declaration"]
        MT["memory_type"]
    end

    subgraph NONE["none — Stateless"]
        N1[No history passed to LLM]
        N2[Each turn is independent]
    end

    subgraph ST["short-term — Active"]
        S1[Rolling window: last 10 msgs]
        S2[Tool results capped at 2000 chars]
        S3[Stored in RuntimeState.messages]
        S4[Passed to LLM each turn]
        S1 --> S3
        S2 --> S3
        S3 --> S4
    end

    subgraph PLAT["platformchat — Per-Chat Sessions"]
        PC1[session key: platform + chat_id]
        PC2[in-process dict: runtime_bridge]
        PC3[isolated history per chat]
        PC1 --> PC2
        PC2 --> PC3
    end

    subgraph LT["long-term — Planned"]
        L1[SQLite or Redis checkpointer]
        L2[Load graph state on session start]
        L3[Auto-save after every node]
        L4[Summarizer compresses old turns]
        L1 --> L2
        L3 --> L1
        L4 --> L2
    end

    subgraph SEM["semantic — Planned"]
        V1[Embed user messages]
        V2[Vector store: Pinecone or pgvector]
        V3[Retrieve top-K memories]
        V4[Inject into system prompt]
        V1 --> V2
        V2 --> V3
        V3 --> V4
    end

    subgraph LLM_IN["LLM Input — assembled per turn"]
        P1[System Prompt: Role + Instructions + Skills]
        P2[Message History: active memory]
        P3[Tool Results: truncated]
    end

    MT -->|none| NONE
    MT -->|short-term| ST
    MT -->|short-term + platformchat| PLAT
    MT -->|long-term| LT
    MT -->|semantic| SEM

    S4 --> P2
    PC3 --> P2
    L2 --> P2
    V4 --> P1

    P1 --> LLM_IN
    P2 --> LLM_IN
    P3 --> LLM_IN
```
