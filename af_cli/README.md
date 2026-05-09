# agentfactory-cli (Go)

CLI tool for validating AgentFactory `.md` agent spec files.

## Install

```bash
cd AgentFactory-CLI/go
go build -o agentfactory .
```

Move the binary to your PATH:

```bash
# macOS / Linux
mv agentfactory /usr/local/bin/

# Windows — move agentfactory.exe to a folder in your PATH
```

## Usage

```bash
agentfactory validate agent.md
agentfactory validate ./agents/weather-forecast.md
```

## Output

```
AgentFactory Spec Validator
weather-forecast.md

✗ Errors (1)
  model.authentication.api_key     api_key is empty — use "${env:YOUR_KEY_VAR}"

⚠ Warnings (2)
  description                      empty — add a one-sentence description
  max_iterations                   not set — will default to 5

✓ Passed (8)
  spec_version                     ✓ 0.3.0
  name                             ✓ "Weather Forecast Agent"
  ...

────────────────────────────────────────────────────────────
✗ Spec has errors  —  1 error(s)  2 warning(s)  8 passed
```

Exit code `0` = valid (errors = 0), exit code `1` = has errors.

## What gets validated

| Check | Severity |
|---|---|
| YAML frontmatter present | error |
| `name` present | error |
| `model.provider` present and known | error / warning |
| `model.name` present | error |
| `model.temperature` in range 0–2 | error |
| API keys use `${env:VAR}` not hardcoded values | warning |
| `# Role` section present and non-empty | error |
| `# Instructions` section present and non-empty | error |
| `execution_mode` is valid value | error |
| `max_iterations` in sensible range | warning |
| Tool transport type and URL/command present | error |
| Interface types are valid | error / warning |
| Skill paths/URLs present | error |
| Output schema JSON is valid | error |
