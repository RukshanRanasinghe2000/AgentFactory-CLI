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
	Line     int // 1-based source line, 0 = unknown
	Col      int // 1-based source column, 0 = unknown
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
// pos is looked up from the spec's position index using the field name,
// with fallback to the nearest parent path.
func (r *Report) emit(cfg *config.Config, spec *parser.Spec, ruleID string, defaultLevel config.RuleLevel, field, msg string) {
	effective := cfg.Resolve(ruleID, defaultLevel)
	if effective == config.LevelOff {
		return
	}

	// Look up position: try exact field, then walk up to parent paths
	pos := lookupPos(spec.Positions, field)

	sev := levelToSeverity(effective)
	r.Results = append(r.Results, Result{
		RuleID:   ruleID,
		Severity: sev,
		Field:    field,
		Message:  msg,
		Line:     pos.Line,
		Col:      pos.Col,
	})
	switch sev {
	case SeverityError:
		r.Errors++
	case SeverityWarning:
		r.Warnings++
	case SeverityInfo:
		r.Infos++
	}
}

// lookupPos tries the exact field key, then progressively strips the last
// path segment to find the nearest parent that has a known position.
func lookupPos(idx parser.FieldIndex, field string) parser.Position {
	key := field
	for key != "" {
		if pos, ok := idx[key]; ok {
			return pos
		}
		// Strip last segment: "interfaces[0].type" → "interfaces[0]" → "interfaces"
		if dot := strings.LastIndex(key, "."); dot >= 0 {
			key = key[:dot]
		} else if bracket := strings.LastIndex(key, "["); bracket >= 0 {
			key = key[:bracket]
		} else {
			break
		}
	}
	return parser.Position{}
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

	checkTypos(r, spec, cfg)
	checkFrontmatter(r, spec, cfg)
	checkModel(r, spec, cfg)
	checkSections(r, spec, cfg)
	checkTools(r, spec, cfg)
	checkInterfaces(r, spec, cfg)
	checkSkills(r, spec, cfg)
	checkOutputSchema(r, spec, cfg)

	return r
}

// ── Typo detection ────────────────────────────────────────────────────────────

func checkTypos(r *Report, s *parser.Spec, cfg *config.Config) {
	for _, t := range s.TypoHints {
		effective := cfg.Resolve("key.typo", config.LevelError)
		if effective == config.LevelOff {
			return
		}
		sev := levelToSeverity(effective)
		field := fmt.Sprintf("line %d, col %d", t.Line, t.Col)
		msg := fmt.Sprintf("unknown key %q — did you mean %q?", t.Found, t.Suggestion)
		r.Results = append(r.Results, Result{
			RuleID:   "key.typo",
			Severity: sev,
			Field:    field,
			Message:  msg,
			Line:     t.Line,
			Col:      t.Col,
		})
		switch sev {
		case SeverityError:
			r.Errors++
		case SeverityWarning:
			r.Warnings++
		case SeverityInfo:
			r.Infos++
		}
	}
}

// ── Frontmatter ───────────────────────────────────────────────────────────────

func checkFrontmatter(r *Report, s *parser.Spec, cfg *config.Config) {
	// spec_version
	if s.SpecVersion == "" {
		r.emit(cfg, s, "spec-version.set", config.LevelWarn,
			"spec_version", "not set — recommended value is \"0.3.0\"")
	} else if !reSpecVersion.MatchString(s.SpecVersion) {
		r.emit(cfg, s, "spec-version.format", config.LevelWarn,
			"spec_version", fmt.Sprintf("unexpected format %q — expected e.g. \"0.3.0\"", s.SpecVersion))
	} else {
		r.emit(cfg, s, "spec-version.set", config.LevelInfo,
			"spec_version", fmt.Sprintf("✓ %s", s.SpecVersion))
	}

	// name
	if strings.TrimSpace(s.Name) == "" {
		r.emit(cfg, s, "name.required", config.LevelError,
			"name", "required — agent must have a name")
	} else if len(s.Name) > 80 {
		r.emit(cfg, s, "name.length", config.LevelWarn,
			"name", "name is very long — keep it under 80 characters")
	} else {
		r.emit(cfg, s, "name.required", config.LevelInfo,
			"name", fmt.Sprintf("✓ %q", s.Name))
	}

	// version
	if s.Version == "" {
		r.emit(cfg, s, "version.set", config.LevelWarn,
			"version", "not set — recommended e.g. \"0.1.0\"")
	} else if !reSemver.MatchString(s.Version) {
		r.emit(cfg, s, "version.semver", config.LevelWarn,
			"version", fmt.Sprintf("%q is not a valid semver — expected e.g. \"1.0.0\"", s.Version))
	} else {
		r.emit(cfg, s, "version.set", config.LevelInfo,
			"version", fmt.Sprintf("✓ %s", s.Version))
	}

	// description
	if strings.TrimSpace(s.Description) == "" {
		r.emit(cfg, s, "description.set", config.LevelWarn,
			"description", "empty — add a one-sentence description of what this agent does")
	} else {
		r.emit(cfg, s, "description.set", config.LevelInfo,
			"description", "✓ present")
	}

	// max_iterations
	if s.MaxIterations == 0 {
		r.emit(cfg, s, "max-iterations.set", config.LevelInfo,
			"max_iterations", "not set — will default to 5")
	} else if s.MaxIterations < 1 {
		r.emit(cfg, s, "max-iterations.min", config.LevelError,
			"max_iterations", "must be at least 1")
	} else if s.MaxIterations > 50 {
		r.emit(cfg, s, "max-iterations.max", config.LevelWarn,
			"max_iterations", fmt.Sprintf("%d is very high — consider 5–30 for most agents", s.MaxIterations))
	} else {
		r.emit(cfg, s, "max-iterations.set", config.LevelInfo,
			"max_iterations", fmt.Sprintf("✓ %d", s.MaxIterations))
	}

	// execution_mode
	switch s.ExecutionMode {
	case "", "sequential":
		r.emit(cfg, s, "execution-mode.valid", config.LevelInfo,
			"execution_mode", fmt.Sprintf("✓ %s", orDefault(s.ExecutionMode, "sequential")))
	case "agentic":
		r.emit(cfg, s, "execution-mode.valid", config.LevelInfo,
			"execution_mode", "✓ agentic")
	default:
		r.emit(cfg, s, "execution-mode.valid", config.LevelError,
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
		r.emit(cfg, s, "model.provider.required", config.LevelError,
			"model.provider", "required — e.g. \"groq\", \"openai\", \"anthropic\"")
	} else if !validProviders[strings.ToLower(m.Provider)] {
		r.emit(cfg, s, "model.provider.known", config.LevelWarn,
			"model.provider",
			fmt.Sprintf("%q is not a known provider — known: openai, groq, anthropic, google, ollama", m.Provider))
	} else {
		r.emit(cfg, s, "model.provider.required", config.LevelInfo,
			"model.provider", fmt.Sprintf("✓ %s", m.Provider))
	}

	if m.Name == "" {
		r.emit(cfg, s, "model.name.required", config.LevelError,
			"model.name", "required — e.g. \"llama-3.3-70b-versatile\", \"gpt-4o\"")
	} else {
		r.emit(cfg, s, "model.name.required", config.LevelInfo,
			"model.name", fmt.Sprintf("✓ %s", m.Name))
	}

	if m.Temperature < 0 || m.Temperature > 2 {
		r.emit(cfg, s, "model.temperature.range", config.LevelError,
			"model.temperature",
			fmt.Sprintf("%.2f is out of range — must be between 0.0 and 2.0", m.Temperature))
	}

	if m.Authentication == nil {
		r.emit(cfg, s, "model.auth.set", config.LevelWarn,
			"model.authentication",
			"not set — agent will rely on server-side env vars for the API key")
	} else {
		auth := m.Authentication
		switch auth.Type {
		case "api-key":
			if auth.APIKey == "" {
				r.emit(cfg, s, "model.auth.api-key.empty", config.LevelError,
					"model.authentication.api_key",
					"api_key is empty — use \"${env:YOUR_KEY_VAR}\" to reference an env variable")
			} else if !reEnvPlaceholder.MatchString(auth.APIKey) {
				r.emit(cfg, s, "model.auth.api-key.hardcoded", config.LevelWarn,
					"model.authentication.api_key",
					"api_key looks like a hardcoded value — use \"${env:VAR_NAME}\" instead")
			} else {
				r.emit(cfg, s, "model.auth.api-key.empty", config.LevelInfo,
					"model.authentication.api_key", fmt.Sprintf("✓ %s", auth.APIKey))
			}
		case "bearer":
			if auth.Token == "" {
				r.emit(cfg, s, "model.auth.bearer.empty", config.LevelError,
					"model.authentication.token", "token is empty for bearer auth")
			}
		case "none", "":
			r.emit(cfg, s, "model.auth.set", config.LevelInfo,
				"model.authentication.type", "no auth configured")
		default:
			r.emit(cfg, s, "model.auth.type.known", config.LevelWarn,
				"model.authentication.type",
				fmt.Sprintf("%q is not a known auth type — use api-key, bearer, or none", auth.Type))
		}
	}
}

// ── Markdown sections ─────────────────────────────────────────────────────────

func checkSections(r *Report, s *parser.Spec, cfg *config.Config) {
	if strings.TrimSpace(s.Role) == "" {
		r.emit(cfg, s, "role.required", config.LevelError,
			"# Role", "section is missing or empty — required for all agents")
	} else if len(s.Role) < 50 {
		r.emit(cfg, s, "role.length", config.LevelWarn,
			"# Role",
			fmt.Sprintf("very short (%d chars) — describe the agent's persona and expertise in detail", len(s.Role)))
	} else if !strings.Contains(strings.ToLower(s.Role), "you are") {
		r.emit(cfg, s, "role.you-are", config.LevelWarn,
			"# Role", "should start with \"You are...\" to set the agent's persona clearly")
	} else {
		r.emit(cfg, s, "role.required", config.LevelInfo,
			"# Role", fmt.Sprintf("✓ %d chars", len(s.Role)))
	}

	if strings.TrimSpace(s.Instructions) == "" {
		r.emit(cfg, s, "instructions.required", config.LevelError,
			"# Instructions", "section is missing or empty — required for all agents")
	} else if len(s.Instructions) < 100 {
		r.emit(cfg, s, "instructions.length", config.LevelWarn,
			"# Instructions",
			fmt.Sprintf("very short (%d chars) — add step-by-step instructions with ## headings", len(s.Instructions)))
	} else {
		r.emit(cfg, s, "instructions.required", config.LevelInfo,
			"# Instructions", fmt.Sprintf("✓ %d chars", len(s.Instructions)))
	}

	if strings.TrimSpace(s.Enforcement) == "" {
		r.emit(cfg, s, "enforcement.set", config.LevelInfo,
			"# Enforcement", "not set — consider adding hard rules the agent must follow")
	} else {
		r.emit(cfg, s, "enforcement.set", config.LevelInfo,
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
		r.emit(cfg, s, "tools.empty", config.LevelInfo, "tools", "tools block is present but empty")
		return
	}

	for i, t := range toolList {
		tool, ok := t.(map[string]interface{})
		if !ok {
			r.emit(cfg, s, "tools.valid", config.LevelError,
				fmt.Sprintf("tools[%d]", i), "tool entry is not a valid object")
			continue
		}

		prefix := fmt.Sprintf("tools[%d]", i)
		name, _ := tool["name"].(string)
		if name != "" {
			prefix = fmt.Sprintf("tools.%s", name)
		}

		if strings.TrimSpace(name) == "" {
			r.emit(cfg, s, "tool.name.required", config.LevelError,
				prefix+".name", "tool must have a name")
		} else {
			r.emit(cfg, s, "tool.name.required", config.LevelInfo,
				prefix+".name", fmt.Sprintf("✓ %q", name))
		}

		transport, hasTransport := tool["transport"].(map[string]interface{})
		if !hasTransport {
			r.emit(cfg, s, "tool.transport.required", config.LevelError,
				prefix+".transport", "transport is required")
			continue
		}

		tType, _ := transport["type"].(string)
		switch tType {
		case "http":
			url, _ := transport["url"].(string)
			if url == "" {
				r.emit(cfg, s, "tool.transport.url.required", config.LevelError,
					prefix+".transport.url", "url is required for HTTP transport")
			} else {
				r.emit(cfg, s, "tool.transport.url.required", config.LevelInfo,
					prefix+".transport.url", fmt.Sprintf("✓ %s", url))
			}
		case "stdio":
			cmd, _ := transport["command"].(string)
			if cmd == "" {
				r.emit(cfg, s, "tool.transport.command.required", config.LevelError,
					prefix+".transport.command", "command is required for stdio transport")
			} else {
				r.emit(cfg, s, "tool.transport.command.required", config.LevelInfo,
					prefix+".transport.command", fmt.Sprintf("✓ %s", cmd))
			}
		case "":
			r.emit(cfg, s, "tool.transport.type.required", config.LevelError,
				prefix+".transport.type", "transport type is required — use \"http\" or \"stdio\"")
		default:
			r.emit(cfg, s, "tool.transport.type.known", config.LevelWarn,
				prefix+".transport.type",
				fmt.Sprintf("%q is not a known transport type — use \"http\" or \"stdio\"", tType))
		}

		if auth, ok := tool["authentication"].(map[string]interface{}); ok {
			authType, _ := auth["type"].(string)
			switch authType {
			case "api-key":
				key, _ := auth["api_key"].(string)
				if key == "" {
					r.emit(cfg, s, "tool.auth.api-key.empty", config.LevelError,
						prefix+".authentication.api_key", "api_key is empty — use \"${env:VAR_NAME}\"")
				} else if !reEnvPlaceholder.MatchString(key) {
					r.emit(cfg, s, "tool.auth.api-key.hardcoded", config.LevelWarn,
						prefix+".authentication.api_key",
						"looks like a hardcoded key — use \"${env:VAR_NAME}\" instead")
				}
			case "bearer":
				token, _ := auth["token"].(string)
				if token == "" {
					r.emit(cfg, s, "tool.auth.bearer.empty", config.LevelError,
						prefix+".authentication.token", "token is empty for bearer auth")
				}
			case "basic":
				user, _ := auth["username"].(string)
				pass, _ := auth["password"].(string)
				if user == "" {
					r.emit(cfg, s, "tool.auth.basic.username", config.LevelError,
						prefix+".authentication.username", "username is required for basic auth")
				}
				if pass == "" {
					r.emit(cfg, s, "tool.auth.basic.password", config.LevelError,
						prefix+".authentication.password", "password is required for basic auth")
				}
			}
		}
	}
}

// ── Interfaces ────────────────────────────────────────────────────────────────

func checkInterfaces(r *Report, s *parser.Spec, cfg *config.Config) {
	if len(s.Interfaces) == 0 {
		r.emit(cfg, s, "interfaces.set", config.LevelInfo,
			"interfaces", "not set — agent will default to consolechat")
		return
	}

	validTypes := map[string]bool{
		"webchat": true, "consolechat": true, "webhook": true,
	}

	for i, iface := range s.Interfaces {
		prefix := fmt.Sprintf("interfaces[%d]", i)
		if iface.Type == "" {
			r.emit(cfg, s, "interface.type.required", config.LevelError,
				prefix+".type", "interface type is required")
		} else if !validTypes[iface.Type] {
			r.emit(cfg, s, "interface.type.known", config.LevelWarn,
				prefix+".type",
				fmt.Sprintf("%q is not a known interface type — use webchat, consolechat, or webhook", iface.Type))
		} else {
			r.emit(cfg, s, "interface.type.required", config.LevelInfo,
				prefix+".type", fmt.Sprintf("✓ %s", iface.Type))
		}

		if iface.Type == "webhook" && iface.Prompt == "" {
			r.emit(cfg, s, "interface.webhook.prompt", config.LevelWarn,
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
				r.emit(cfg, s, "skill.local.path", config.LevelError,
					prefix+".path", "path is required for local skills")
			} else {
				r.emit(cfg, s, "skill.local.path", config.LevelInfo,
					prefix+".path", fmt.Sprintf("✓ %s", skill.Path))
			}
		case "remote":
			if skill.URL == "" {
				r.emit(cfg, s, "skill.remote.url", config.LevelError,
					prefix+".url", "url is required for remote skills")
			} else {
				r.emit(cfg, s, "skill.remote.url", config.LevelInfo,
					prefix+".url", fmt.Sprintf("✓ %s", skill.URL))
			}
		case "":
			r.emit(cfg, s, "skill.type.required", config.LevelError,
				prefix+".type", "skill type is required — use \"local\" or \"remote\"")
		default:
			r.emit(cfg, s, "skill.type.known", config.LevelWarn,
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
		r.emit(cfg, s, "output-schema.valid-json", config.LevelError,
			"# Output Schema",
			fmt.Sprintf("JSON in output schema is not valid: %v", err))
	} else {
		r.emit(cfg, s, "output-schema.valid-json", config.LevelInfo,
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
