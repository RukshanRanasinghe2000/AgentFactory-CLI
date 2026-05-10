// Package validator checks an AgentFactory spec for correctness.
// Every rule has a stable ID that can be overridden via .afvalidate.toml.
package validator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/agentfactory/cli/config"
	"github.com/agentfactory/cli/parser"
)

// Severity levels for validation results.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Result is a single validation finding.
type Result struct {
	RuleID   string
	Severity Severity
	Field    string
	Message  string
}

// Report holds all findings from a validation run.
type Report struct {
	File     string
	Results  []Result
	Errors   int
	Warnings int
	Infos    int
}

// emit adds a result after applying any config override for ruleID.
// If the rule is set to "off" in config it is silently skipped.
// If the rule level is overridden the new severity is used instead.
func (r *Report) emit(cfg *config.Config, ruleID string, defaultLevel config.RuleLevel, field, msg string) {
	effective := cfg.Resolve(ruleID, defaultLevel)
	if effective == config.LevelOff {
		return
	}

	sev := levelToSeverity(effective)
	r.Results = append(r.Results, Result{RuleID: ruleID, Severity: sev, Field: field, Message: msg})
	switch sev {
	case SeverityError:
		r.Errors++
	case SeverityWarning:
		r.Warnings++
	case SeverityInfo:
		r.Infos++
	}
}

func levelToSeverity(l config.RuleLevel) Severity {
	switch l {
	case config.LevelError:
		return SeverityError
	case config.LevelWarn:
		return SeverityWarning
	default:
		return SeverityInfo
	}
}

var (
	reEnvPlaceholder = regexp.MustCompile(`^\$\{env:.+\}$`)
	reSemver         = regexp.MustCompile(`^\d+\.\d+\.\d+`)
	reSpecVersion    = regexp.MustCompile(`^0\.[0-9]+\.[0-9]+$`)
)

// Validate runs all checks on a parsed spec and returns a Report.
func Validate(file string, spec *parser.Spec, cfg *config.Config) *Report {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	r := &Report{File: file}

	checkFrontmatter(r, spec, cfg)
	checkModel(r, spec, cfg)
	checkSections(r, spec, cfg)
	checkTools(r, spec, cfg)
	checkInterfaces(r, spec, cfg)
	checkSkills(r, spec, cfg)
	checkOutputSchema(r, spec, cfg)

	return r
}

// ── Frontmatter ───────────────────────────────────────────────────────────────

func checkFrontmatter(r *Report, s *parser.Spec, cfg *config.Config) {
	// spec_version
	if s.SpecVersion == "" {
		r.emit(cfg, "spec-version.set", config.LevelWarn,
			"spec_version", "not set — recommended value is \"0.3.0\"")
	} else if !reSpecVersion.MatchString(s.SpecVersion) {
		r.emit(cfg, "spec-version.format", config.LevelWarn,
			"spec_version", fmt.Sprintf("unexpected format %q — expected e.g. \"0.3.0\"", s.SpecVersion))
	} else {
		r.emit(cfg, "spec-version.set", config.LevelInfo,
			"spec_version", fmt.Sprintf("✓ %s", s.SpecVersion))
	}

	// name
	if strings.TrimSpace(s.Name) == "" {
		r.emit(cfg, "name.required", config.LevelError,
			"name", "required — agent must have a name")
	} else if len(s.Name) > 80 {
		r.emit(cfg, "name.length", config.LevelWarn,
			"name", "name is very long — keep it under 80 characters")
	} else {
		r.emit(cfg, "name.required", config.LevelInfo,
			"name", fmt.Sprintf("✓ %q", s.Name))
	}

	// version
	if s.Version == "" {
		r.emit(cfg, "version.set", config.LevelWarn,
			"version", "not set — recommended e.g. \"0.1.0\"")
	} else if !reSemver.MatchString(s.Version) {
		r.emit(cfg, "version.semver", config.LevelWarn,
			"version", fmt.Sprintf("%q is not a valid semver — expected e.g. \"1.0.0\"", s.Version))
	} else {
		r.emit(cfg, "version.set", config.LevelInfo,
			"version", fmt.Sprintf("✓ %s", s.Version))
	}

	// description
	if strings.TrimSpace(s.Description) == "" {
		r.emit(cfg, "description.set", config.LevelWarn,
			"description", "empty — add a one-sentence description of what this agent does")
	} else {
		r.emit(cfg, "description.set", config.LevelInfo,
			"description", "✓ present")
	}

	// max_iterations
	if s.MaxIterations == 0 {
		r.emit(cfg, "max-iterations.set", config.LevelInfo,
			"max_iterations", "not set — will default to 5")
	} else if s.MaxIterations < 1 {
		r.emit(cfg, "max-iterations.min", config.LevelError,
			"max_iterations", "must be at least 1")
	} else if s.MaxIterations > 50 {
		r.emit(cfg, "max-iterations.max", config.LevelWarn,
			"max_iterations", fmt.Sprintf("%d is very high — consider 5–30 for most agents", s.MaxIterations))
	} else {
		r.emit(cfg, "max-iterations.set", config.LevelInfo,
			"max_iterations", fmt.Sprintf("✓ %d", s.MaxIterations))
	}

	// execution_mode
	switch s.ExecutionMode {
	case "", "sequential":
		r.emit(cfg, "execution-mode.valid", config.LevelInfo,
			"execution_mode", fmt.Sprintf("✓ %s", orDefault(s.ExecutionMode, "sequential")))
	case "agentic":
		r.emit(cfg, "execution-mode.valid", config.LevelInfo,
			"execution_mode", "✓ agentic")
	default:
		r.emit(cfg, "execution-mode.valid", config.LevelError,
			"execution_mode",
			fmt.Sprintf("%q is not valid — must be \"sequential\" or \"agentic\"", s.ExecutionMode))
	}
}

// ── Model ─────────────────────────────────────────────────────────────────────

func checkModel(r *Report, s *parser.Spec, cfg *config.Config) {
	m := s.Model

	validProviders := map[string]bool{
		"openai": true, "groq": true, "anthropic": true,
		"google": true, "ollama": true,
	}

	if m.Provider == "" {
		r.emit(cfg, "model.provider.required", config.LevelError,
			"model.provider", "required — e.g. \"groq\", \"openai\", \"anthropic\"")
	} else if !validProviders[strings.ToLower(m.Provider)] {
		r.emit(cfg, "model.provider.known", config.LevelWarn,
			"model.provider",
			fmt.Sprintf("%q is not a known provider — known: openai, groq, anthropic, google, ollama", m.Provider))
	} else {
		r.emit(cfg, "model.provider.required", config.LevelInfo,
			"model.provider", fmt.Sprintf("✓ %s", m.Provider))
	}

	if m.Name == "" {
		r.emit(cfg, "model.name.required", config.LevelError,
			"model.name", "required — e.g. \"llama-3.3-70b-versatile\", \"gpt-4o\"")
	} else {
		r.emit(cfg, "model.name.required", config.LevelInfo,
			"model.name", fmt.Sprintf("✓ %s", m.Name))
	}

	if m.Temperature < 0 || m.Temperature > 2 {
		r.emit(cfg, "model.temperature.range", config.LevelError,
			"model.temperature",
			fmt.Sprintf("%.2f is out of range — must be between 0.0 and 2.0", m.Temperature))
	}

	if m.Authentication == nil {
		r.emit(cfg, "model.auth.set", config.LevelWarn,
			"model.authentication",
			"not set — agent will rely on server-side env vars for the API key")
	} else {
		auth := m.Authentication
		switch auth.Type {
		case "api-key":
			if auth.APIKey == "" {
				r.emit(cfg, "model.auth.api-key.empty", config.LevelError,
					"model.authentication.api_key",
					"api_key is empty — use \"${env:YOUR_KEY_VAR}\" to reference an env variable")
			} else if !reEnvPlaceholder.MatchString(auth.APIKey) {
				r.emit(cfg, "model.auth.api-key.hardcoded", config.LevelWarn,
					"model.authentication.api_key",
					"api_key looks like a hardcoded value — use \"${env:VAR_NAME}\" instead")
			} else {
				r.emit(cfg, "model.auth.api-key.empty", config.LevelInfo,
					"model.authentication.api_key", fmt.Sprintf("✓ %s", auth.APIKey))
			}
		case "bearer":
			if auth.Token == "" {
				r.emit(cfg, "model.auth.bearer.empty", config.LevelError,
					"model.authentication.token", "token is empty for bearer auth")
			}
		case "none", "":
			r.emit(cfg, "model.auth.set", config.LevelInfo,
				"model.authentication.type", "no auth configured")
		default:
			r.emit(cfg, "model.auth.type.known", config.LevelWarn,
				"model.authentication.type",
				fmt.Sprintf("%q is not a known auth type — use api-key, bearer, or none", auth.Type))
		}
	}
}

// ── Markdown sections ─────────────────────────────────────────────────────────

func checkSections(r *Report, s *parser.Spec, cfg *config.Config) {
	// Role
	if strings.TrimSpace(s.Role) == "" {
		r.emit(cfg, "role.required", config.LevelError,
			"# Role", "section is missing or empty — required for all agents")
	} else if len(s.Role) < 50 {
		r.emit(cfg, "role.length", config.LevelWarn,
			"# Role",
			fmt.Sprintf("very short (%d chars) — describe the agent's persona and expertise in detail", len(s.Role)))
	} else if !strings.Contains(strings.ToLower(s.Role), "you are") {
		r.emit(cfg, "role.you-are", config.LevelWarn,
			"# Role", "should start with \"You are...\" to set the agent's persona clearly")
	} else {
		r.emit(cfg, "role.required", config.LevelInfo,
			"# Role", fmt.Sprintf("✓ %d chars", len(s.Role)))
	}

	// Instructions
	if strings.TrimSpace(s.Instructions) == "" {
		r.emit(cfg, "instructions.required", config.LevelError,
			"# Instructions", "section is missing or empty — required for all agents")
	} else if len(s.Instructions) < 100 {
		r.emit(cfg, "instructions.length", config.LevelWarn,
			"# Instructions",
			fmt.Sprintf("very short (%d chars) — add step-by-step instructions with ## headings", len(s.Instructions)))
	} else {
		r.emit(cfg, "instructions.required", config.LevelInfo,
			"# Instructions", fmt.Sprintf("✓ %d chars", len(s.Instructions)))
	}

	// Enforcement
	if strings.TrimSpace(s.Enforcement) == "" {
		r.emit(cfg, "enforcement.set", config.LevelInfo,
			"# Enforcement", "not set — consider adding hard rules the agent must follow")
	} else {
		r.emit(cfg, "enforcement.set", config.LevelInfo,
			"# Enforcement", fmt.Sprintf("✓ %d chars", len(s.Enforcement)))
	}
}

// ── Tools ─────────────────────────────────────────────────────────────────────

func checkTools(r *Report, s *parser.Spec, cfg *config.Config) {
	if s.Tools == nil {
		return
	}

	var toolList []interface{}
	switch v := s.Tools.(type) {
	case []interface{}:
		toolList = v
	case map[string]interface{}:
		if mcp, ok := v["mcp"]; ok {
			if list, ok := mcp.([]interface{}); ok {
				toolList = list
			}
		}
	}

	if len(toolList) == 0 {
		r.emit(cfg, "tools.empty", config.LevelInfo, "tools", "tools block is present but empty")
		return
	}

	for i, t := range toolList {
		tool, ok := t.(map[string]interface{})
		if !ok {
			r.emit(cfg, "tools.valid", config.LevelError,
				fmt.Sprintf("tools[%d]", i), "tool entry is not a valid object")
			continue
		}

		prefix := fmt.Sprintf("tools[%d]", i)
		name, _ := tool["name"].(string)
		if name != "" {
			prefix = fmt.Sprintf("tools.%s", name)
		}

		if strings.TrimSpace(name) == "" {
			r.emit(cfg, "tool.name.required", config.LevelError,
				prefix+".name", "tool must have a name")
		} else {
			r.emit(cfg, "tool.name.required", config.LevelInfo,
				prefix+".name", fmt.Sprintf("✓ %q", name))
		}

		transport, hasTransport := tool["transport"].(map[string]interface{})
		if !hasTransport {
			r.emit(cfg, "tool.transport.required", config.LevelError,
				prefix+".transport", "transport is required")
			continue
		}

		tType, _ := transport["type"].(string)
		switch tType {
		case "http":
			url, _ := transport["url"].(string)
			if url == "" {
				r.emit(cfg, "tool.transport.url.required", config.LevelError,
					prefix+".transport.url", "url is required for HTTP transport")
			} else {
				r.emit(cfg, "tool.transport.url.required", config.LevelInfo,
					prefix+".transport.url", fmt.Sprintf("✓ %s", url))
			}
		case "stdio":
			cmd, _ := transport["command"].(string)
			if cmd == "" {
				r.emit(cfg, "tool.transport.command.required", config.LevelError,
					prefix+".transport.command", "command is required for stdio transport")
			} else {
				r.emit(cfg, "tool.transport.command.required", config.LevelInfo,
					prefix+".transport.command", fmt.Sprintf("✓ %s", cmd))
			}
		case "":
			r.emit(cfg, "tool.transport.type.required", config.LevelError,
				prefix+".transport.type", "transport type is required — use \"http\" or \"stdio\"")
		default:
			r.emit(cfg, "tool.transport.type.known", config.LevelWarn,
				prefix+".transport.type",
				fmt.Sprintf("%q is not a known transport type — use \"http\" or \"stdio\"", tType))
		}

		if auth, ok := tool["authentication"].(map[string]interface{}); ok {
			authType, _ := auth["type"].(string)
			switch authType {
			case "api-key":
				key, _ := auth["api_key"].(string)
				if key == "" {
					r.emit(cfg, "tool.auth.api-key.empty", config.LevelError,
						prefix+".authentication.api_key", "api_key is empty — use \"${env:VAR_NAME}\"")
				} else if !reEnvPlaceholder.MatchString(key) {
					r.emit(cfg, "tool.auth.api-key.hardcoded", config.LevelWarn,
						prefix+".authentication.api_key",
						"looks like a hardcoded key — use \"${env:VAR_NAME}\" instead")
				}
			case "bearer":
				token, _ := auth["token"].(string)
				if token == "" {
					r.emit(cfg, "tool.auth.bearer.empty", config.LevelError,
						prefix+".authentication.token", "token is empty for bearer auth")
				}
			case "basic":
				user, _ := auth["username"].(string)
				pass, _ := auth["password"].(string)
				if user == "" {
					r.emit(cfg, "tool.auth.basic.username", config.LevelError,
						prefix+".authentication.username", "username is required for basic auth")
				}
				if pass == "" {
					r.emit(cfg, "tool.auth.basic.password", config.LevelError,
						prefix+".authentication.password", "password is required for basic auth")
				}
			}
		}
	}
}

// ── Interfaces ────────────────────────────────────────────────────────────────

func checkInterfaces(r *Report, s *parser.Spec, cfg *config.Config) {
	if len(s.Interfaces) == 0 {
		r.emit(cfg, "interfaces.set", config.LevelInfo,
			"interfaces", "not set — agent will default to consolechat")
		return
	}

	validTypes := map[string]bool{
		"webchat": true, "consolechat": true, "webhook": true,
	}

	for i, iface := range s.Interfaces {
		prefix := fmt.Sprintf("interfaces[%d]", i)
		if iface.Type == "" {
			r.emit(cfg, "interface.type.required", config.LevelError,
				prefix+".type", "interface type is required")
		} else if !validTypes[iface.Type] {
			r.emit(cfg, "interface.type.known", config.LevelWarn,
				prefix+".type",
				fmt.Sprintf("%q is not a known interface type — use webchat, consolechat, or webhook", iface.Type))
		} else {
			r.emit(cfg, "interface.type.required", config.LevelInfo,
				prefix+".type", fmt.Sprintf("✓ %s", iface.Type))
		}

		if iface.Type == "webhook" && iface.Prompt == "" {
			r.emit(cfg, "interface.webhook.prompt", config.LevelWarn,
				prefix+".prompt",
				"webhook interface should have a prompt template with ${http:payload.*} placeholders")
		}
	}
}

// ── Skills ────────────────────────────────────────────────────────────────────

func checkSkills(r *Report, s *parser.Spec, cfg *config.Config) {
	for i, skill := range s.Skills {
		prefix := fmt.Sprintf("skills[%d]", i)
		switch skill.Type {
		case "local":
			if skill.Path == "" {
				r.emit(cfg, "skill.local.path", config.LevelError,
					prefix+".path", "path is required for local skills")
			} else {
				r.emit(cfg, "skill.local.path", config.LevelInfo,
					prefix+".path", fmt.Sprintf("✓ %s", skill.Path))
			}
		case "remote":
			if skill.URL == "" {
				r.emit(cfg, "skill.remote.url", config.LevelError,
					prefix+".url", "url is required for remote skills")
			} else {
				r.emit(cfg, "skill.remote.url", config.LevelInfo,
					prefix+".url", fmt.Sprintf("✓ %s", skill.URL))
			}
		case "":
			r.emit(cfg, "skill.type.required", config.LevelError,
				prefix+".type", "skill type is required — use \"local\" or \"remote\"")
		default:
			r.emit(cfg, "skill.type.known", config.LevelWarn,
				prefix+".type",
				fmt.Sprintf("%q is not a known skill type — use \"local\" or \"remote\"", skill.Type))
		}
	}
}

// ── Output Schema ─────────────────────────────────────────────────────────────

func checkOutputSchema(r *Report, s *parser.Spec, cfg *config.Config) {
	if s.OutputSchema == "" {
		return
	}
	var js interface{}
	if err := json.Unmarshal([]byte(s.OutputSchema), &js); err != nil {
		r.emit(cfg, "output-schema.valid-json", config.LevelError,
			"# Output Schema",
			fmt.Sprintf("JSON in output schema is not valid: %v", err))
	} else {
		r.emit(cfg, "output-schema.valid-json", config.LevelInfo,
			"# Output Schema", "✓ valid JSON")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
