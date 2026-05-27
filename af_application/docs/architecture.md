# AgentFactory-CLI ( Abstract Architecture )

## 1. Top-Level: User to CLI to Python

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

```mermaid
graph TD
    subgraph DECL["Spec Declaration"]
        MT["memory_type: none / short-term / long-term / semantic"]
    end

    subgraph NONE["none - Stateless"]
        N1[No history passed to LLM]
        N2[Each turn is independent]
    end

    subgraph ST["short-term - Current Implementation"]
        S1[Rolling window - last 10 messages kept]
        S2[Tool results truncated to 2000 chars]
        S3[messages list stored in RuntimeState]
        S4[Passed to LLM on every turn]
        S1 --> S3
        S2 --> S3
        S3 --> S4
    end

    subgraph PLAT["platformchat - Per-Chat Sessions"]
        PC1[session key = platform + chat_id]
        PC2[in-process dict in runtime_bridge.py]
        PC3[isolated history per Telegram chat]
        PC1 --> PC2
        PC2 --> PC3
    end

    subgraph LT["long-term - Planned"]
        L1[Persistent store - SQLite or Redis]
        L2[Load history on session start]
        L3[Save each turn on session end]
        L4[Summarizer compresses old turns]
        L1 --> L2
        L3 --> L1
        L4 --> L2
    end

    subgraph SEM["semantic - Planned"]
        V1[Embed user messages]
        V2[Vector store - Pinecone or pgvector]
        V3[Retrieve top-K relevant memories]
        V4[Inject into system prompt]
        V1 --> V2
        V2 --> V3
        V3 --> V4
    end

    subgraph LLM_IN["LLM Input - assembled per turn"]
        P1[System Prompt - Role + Instructions + Skills]
        P2[Message History - from active memory type]
        P3[Tool Results - truncated]
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
