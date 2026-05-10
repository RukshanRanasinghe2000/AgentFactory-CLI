---
# Rule: model.auth.set
# Triggers: warning — model.authentication block is absent
# Expected: SeverityWarning on field "model.authentication"
spec_version: "0.3.0"
name: "Test Agent"
description: "A test agent."
version: "0.1.0"
max_iterations: 5
execution_mode: "sequential"
model:
  provider: "groq"
  name: "llama-3.3-70b-versatile"
---

# Role

You are a helpful test agent for validating rules.

---

# Instructions

## Step 1
Do the first thing carefully.

## Step 2
Do the second thing carefully and completely.
