---
# Rule: model.provider.known
# Triggers: warning — model.provider is not a known provider
# Expected: SeverityWarning on field "model.provider"
spec_version: "0.3.0"
name: "Test Agent"
description: "A test agent."
version: "0.1.0"
max_iterations: 5
execution_mode: "sequential"
model:
  provider: "mistral"
  name: "mistral-large"
  authentication:
    type: "api-key"
    api_key: "${env:MISTRAL_API_KEY}"
---

# Role

You are a helpful test agent for validating rules.

---

# Instructions

## Step 1
Do the first thing carefully.

## Step 2
Do the second thing carefully and completely.
