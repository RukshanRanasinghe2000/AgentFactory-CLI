---
# Rule: model.temperature.range
# Triggers: error — model.temperature is outside 0.0–2.0
# Expected: SeverityError on field "model.temperature"
spec_version: "0.3.0"
name: "Test Agent"
description: "A test agent."
version: "0.1.0"
max_iterations: 5
execution_mode: "sequential"
model:
  provider: "groq"
  name: "llama-3.3-70b-versatile"
  temperature: 3.5
  authentication:
    type: "api-key"
    api_key: "${env:GROQ_API_KEY}"
---

# Role

You are a helpful test agent for validating rules.

---

# Instructions

## Step 1
Do the first thing carefully.

## Step 2
Do the second thing carefully and completely.
