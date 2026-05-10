---
# Rule: instructions.length
# Triggers: warning — # Instructions content is present but under 100 characters
# Expected: SeverityWarning on field "# Instructions"
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
---

# Role

You are a helpful test agent for validating rules.

---

# Instructions

Do the task.
