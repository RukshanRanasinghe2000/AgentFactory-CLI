---
spec_version: "0.3.0"
name: "Test Agent"
version: "not-semver"
max_iterations: 5
model:
  provider: "groq"
  name: "llama-3.3-70b-versatile"
  authentication:
    type: "api-key"
    api_key: "${env:GROQ_API_KEY}"
---

# Role

A helpful assistant that answers questions.

---

# Instructions

## Step 1
Answer the user's question clearly and concisely.

## Step 2
If you don't know, say so honestly.
