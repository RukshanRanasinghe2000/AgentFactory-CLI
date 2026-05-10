---
# Rule: role.you-are
# Triggers: warning — # Role is 50+ chars but does not contain "you are"
# Expected: SeverityWarning on field "# Role"
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

An expert assistant that helps with various tasks and provides detailed answers to questions.

---

# Instructions

## Step 1
Do the first thing carefully.

## Step 2
Do the second thing carefully and completely.
