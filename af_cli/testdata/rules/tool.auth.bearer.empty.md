---
# Rule: tool.auth.bearer.empty
# Triggers: error — tool auth type is bearer but token is empty
# Expected: SeverityError on field "tools.my_tool.authentication.token"
spec_version: "0.3.0"
name: "Test Agent"
description: "A test agent."
version: "0.1.0"
max_iterations: 5
execution_mode: "sequential"
model:
  provider: "groq"
  name: "llama-3.3-70b-versatile"
  authentication:
    type: "api-key"
    api_key: "${env:GROQ_API_KEY}"
tools:
  mcp:
    - name: "my_tool"
      transport:
        type: "http"
        url: "https://api.example.com"
      authentication:
        type: "bearer"
        token: ""
---

# Role

You are a helpful test agent for validating rules.

---

# Instructions

## Step 1
Do the first thing carefully.

## Step 2
Do the second thing carefully and completely.
