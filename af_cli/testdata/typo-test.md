---
spec_versoin: "0.3.0"
name: "Typo Test Agent"
description: "Tests typo detection."
version: "0.1.0"
max_iterations: 5
execution_mode: "sequential"
model:
  provder: "groq"
  name: "llama-3.3-70b-versatile"
  authenticaton:
    type: "api-key"
    api_key: "${env:GROQ_API_KEY}"
interfaces:
  - tpye: "consolechat"
---

# Role

You are a helpful test agent for validating typo detection rules.

---

# Instructions

## Step 1
Do the first thing carefully.

## Step 2
Do the second thing carefully and completely.
