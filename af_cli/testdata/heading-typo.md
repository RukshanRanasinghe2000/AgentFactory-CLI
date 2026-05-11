---
spec_version: "0.3.0"
name: "Heading Typo Test"
description: "Tests section heading typo detection."
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

# Rolee

You are a helpful test agent for validating heading typo detection rules.

---

# Instrutions

## Step 1
Do the first thing carefully.

## Step 2
Do the second thing carefully and completely.

---

# Enforcment

- Always follow the rules.
