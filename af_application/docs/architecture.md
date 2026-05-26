# AgentFactory-CLI ( Abstract Architecture )

## 1. Top-Level: User to CLI to Python

```mermaid
graph TD
    U([User]) -->|agentfactory vali agent.md| CLI_VALI[Go CLI - vali command]
    U -->|agentfactory run agent.md| CLI_RUN[Go CLI - run command]

    CLI_VALI -->|parse and validate| PARSER[lexer + parser + validator]
    PARSER -->|validation report| CLI_VALI
    CLI_VALI -->|results| U

    CLI_RUN -->|stdin JSON phase=load| RB[runtime_bridge.py]
    CLI_RUN -->|stdin JSON phase=chat| RB

    RB -->|load_spec| SL[spec_loader.py]
    SL -->|AgentSpec dict| RB

    RB -->|invoke| RG[runtime_graph]
    RG -->|assistant_response| RB
    RB -->|JSON response| CLI_RUN
    CLI_RUN -->|response| U
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

## 4. Validation Flow

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

## 5. Tool Execution: HTTP vs stdio

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

## 6. Memory Architecture

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
    MT -->|long-term| LT
    MT -->|semantic| SEM

    S4 --> P2
    L2 --> P2
    V4 --> P1

    P1 --> LLM_IN
    P2 --> LLM_IN
    P3 --> LLM_IN
```
