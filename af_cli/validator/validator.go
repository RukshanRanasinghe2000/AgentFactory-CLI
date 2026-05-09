// Package validator checks an AgentFactory spec for correctness.
// Produces structured results with severity levels: error, warning, info.
package validator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

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
	Severity Severity
	Field    string
	Message  string
}

// Report holds all findings from a validation run.
type Report struct {
	File    string
	Results []Result
	Errors  int
	Warnings int
	Infos   int
}

func (r *Report) add(sev Severity, field, msg string) {
	r.Results = append(r.Results, Result{sev, field, msg})
	switch sev {
	case SeverityError:
		r.Errors++
	case SeverityWarning:
		r.Warnings++
	case SeverityInfo:
		r.Infos++
	}
}

var (
	reEnvPlaceholder = regexp.MustCompile(`^\$\{env:.+\}$`)
	reSemver         = regexp.MustCompile(`^\d+\.\d+\.\d+`)
	reSpecVersion    = regexp.MustCompile(`^0\.[0-9]+\.[0-9]+$`)
)

// Validate runs all checks on a parsed spec and returns a Report.
func Validate(file string, spec *parser.Spec) *Report {
	r := &Report{File: file}

	checkFrontmatter(r, spec)
	checkModel(r, spec)
	checkSections(r, spec)
	checkTools(r, spec)
	checkInterfaces(r, spec)
	checkSkills(r, spec)
	checkOutputSchema(r, spec)

	return r
}

// Frontmatter 

func checkFrontmatter(r *Report, s *parser.Spec) {
	// spec_version
	if s.SpecVersion == "" {
		r.add(SeverityWarning, "spec_version", "not set — recommended value is \"0.3.0\"")
	} else if !reSpecVersion.MatchString(s.SpecVersion) {
		r.add(SeverityWarning, "spec_version", fmt.Sprintf("unexpected format %q — expected e.g. \"0.3.0\"", s.SpecVersion))
	} else {
		r.add(SeverityInfo, "spec_version", fmt.Sprintf("✓ %s", s.SpecVersion))
	}

	// name
	if strings.TrimSpace(s.Name) == "" {
		r.add(SeverityError, "name", "required — agent must have a name")
	} else if len(s.Name) > 80 {
		r.add(SeverityWarning, "name", "name is very long — keep it under 80 characters")
	} else {
		r.add(SeverityInfo, "name", fmt.Sprintf("✓ %q", s.Name))
	}

	// version
	if s.Version == "" {
		r.add(SeverityWarning, "version", "not set — recommended e.g. \"0.1.0\"")
	} else if !reSemver.MatchString(s.Version) {
		r.add(SeverityWarning, "version", fmt.Sprintf("%q is not a valid semver — expected e.g. \"1.0.0\"", s.Version))
	} else {
		r.add(SeverityInfo, "version", fmt.Sprintf("✓ %s", s.Version))
	}

	// description
	if strings.TrimSpace(s.Description) == "" {
		r.add(SeverityWarning, "description", "empty — add a one-sentence description of what this agent does")
	} else {
		r.add(SeverityInfo, "description", "✓ present")
	}

	// max_iterations
	if s.MaxIterations == 0 {
		r.add(SeverityInfo, "max_iterations", "not set — will default to 5")
	} else if s.MaxIterations < 1 {
		r.add(SeverityError, "max_iterations", "must be at least 1")
	} else if s.MaxIterations > 50 {
		r.add(SeverityWarning, "max_iterations", fmt.Sprintf("%d is very high — consider 5–30 for most agents", s.MaxIterations))
	} else {
		r.add(SeverityInfo, "max_iterations", fmt.Sprintf("✓ %d", s.MaxIterations))
	}

	// execution_mode
	switch s.ExecutionMode {
	case "", "sequential":
		r.add(SeverityInfo, "execution_mode", fmt.Sprintf("✓ %s", orDefault(s.ExecutionMode, "sequential")))
	case "agentic":
		r.add(SeverityInfo, "execution_mode", "✓ agentic")
	default:
		r.add(SeverityError, "execution_mode",
			fmt.Sprintf("%q is not valid — must be \"sequential\" or \"agentic\"", s.ExecutionMode))
	}
}

// Model 

func checkModel(r *Report, s *parser.Spec) {
	m := s.Model

	// provider
	validProviders := map[string]bool{
		"openai": true, "groq": true, "anthropic": true,
		"google": true, "ollama": true,
	}
	if m.Provider == "" {
		r.add(SeverityError, "model.provider", "required — e.g. \"groq\", \"openai\", \"anthropic\"")
	} else if !validProviders[strings.ToLower(m.Provider)] {
		r.add(SeverityWarning, "model.provider",
			fmt.Sprintf("%q is not a known provider — known: openai, groq, anthropic, google, ollama", m.Provider))
	} else {
		r.add(SeverityInfo, "model.provider", fmt.Sprintf("✓ %s", m.Provider))
	}

	// name
	if m.Name == "" {
		r.add(SeverityError, "model.name", "required — e.g. \"llama-3.3-70b-versatile\", \"gpt-4o\"")
	} else {
		r.add(SeverityInfo, "model.name", fmt.Sprintf("✓ %s", m.Name))
	}

	// temperature
	if m.Temperature < 0 || m.Temperature > 2 {
		r.add(SeverityError, "model.temperature",
			fmt.Sprintf("%.2f is out of range — must be between 0.0 and 2.0", m.Temperature))
	}

	// authentication
	if m.Authentication == nil {
		r.add(SeverityWarning, "model.authentication",
			"not set — agent will rely on server-side env vars for the API key")
	} else {
		auth := m.Authentication
		switch auth.Type {
		case "api-key":
			if auth.APIKey == "" {
				r.add(SeverityError, "model.authentication.api_key",
					"api_key is empty — use \"${env:YOUR_KEY_VAR}\" to reference an env variable")
			} else if !reEnvPlaceholder.MatchString(auth.APIKey) {
				r.add(SeverityWarning, "model.authentication.api_key",
					"api_key looks like a hardcoded value — use \"${env:VAR_NAME}\" instead")
			} else {
				r.add(SeverityInfo, "model.authentication.api_key", fmt.Sprintf("✓ %s", auth.APIKey))
			}
		case "bearer":
			if auth.Token == "" {
				r.add(SeverityError, "model.authentication.token", "token is empty for bearer auth")
			}
		case "none", "":
			r.add(SeverityInfo, "model.authentication.type", "no auth configured")
		default:
			r.add(SeverityWarning, "model.authentication.type",
				fmt.Sprintf("%q is not a known auth type — use api-key, bearer, or none", auth.Type))
		}
	}
}

// Markdown sections 

func checkSections(r *Report, s *parser.Spec) {
	// Role
	if strings.TrimSpace(s.Role) == "" {
		r.add(SeverityError, "# Role", "section is missing or empty — required for all agents")
	} else if len(s.Role) < 50 {
		r.add(SeverityWarning, "# Role",
			fmt.Sprintf("very short (%d chars) — describe the agent's persona and expertise in detail", len(s.Role)))
	} else if !strings.Contains(strings.ToLower(s.Role), "you are") {
		r.add(SeverityWarning, "# Role", "should start with \"You are...\" to set the agent's persona clearly")
	} else {
		r.add(SeverityInfo, "# Role", fmt.Sprintf("✓ %d chars", len(s.Role)))
	}

	// Instructions
	if strings.TrimSpace(s.Instructions) == "" {
		r.add(SeverityError, "# Instructions", "section is missing or empty — required for all agents")
	} else if len(s.Instructions) < 100 {
		r.add(SeverityWarning, "# Instructions",
			fmt.Sprintf("very short (%d chars) — add step-by-step instructions with ## headings", len(s.Instructions)))
	} else {
		r.add(SeverityInfo, "# Instructions", fmt.Sprintf("✓ %d chars", len(s.Instructions)))
	}

	// Enforcement (optional but recommended)
	if strings.TrimSpace(s.Enforcement) == "" {
		r.add(SeverityInfo, "# Enforcement", "not set — consider adding hard rules the agent must follow")
	} else {
		r.add(SeverityInfo, "# Enforcement", fmt.Sprintf("✓ %d chars", len(s.Enforcement)))
	}
}

// Tools 

func checkTools(r *Report, s *parser.Spec) {
	if s.Tools == nil {
		return // tools are optional
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
		r.add(SeverityInfo, "tools", "tools block is present but empty")
		return
	}

	for i, t := range toolList {
		tool, ok := t.(map[string]interface{})
		if !ok {
			r.add(SeverityError, fmt.Sprintf("tools[%d]", i), "tool entry is not a valid object")
			continue
		}

		prefix := fmt.Sprintf("tools[%d]", i)
		name, _ := tool["name"].(string)
		if name != "" {
			prefix = fmt.Sprintf("tools.%s", name)
		}

		// name
		if strings.TrimSpace(name) == "" {
			r.add(SeverityError, prefix+".name", "tool must have a name")
		} else {
			r.add(SeverityInfo, prefix+".name", fmt.Sprintf("✓ %q", name))
		}

		// transport
		transport, hasTransport := tool["transport"].(map[string]interface{})
		if !hasTransport {
			r.add(SeverityError, prefix+".transport", "transport is required")
			continue
		}

		tType, _ := transport["type"].(string)
		switch tType {
		case "http":
			url, _ := transport["url"].(string)
			if url == "" {
				r.add(SeverityError, prefix+".transport.url", "url is required for HTTP transport")
			} else {
				r.add(SeverityInfo, prefix+".transport.url", fmt.Sprintf("✓ %s", url))
			}
		case "stdio":
			cmd, _ := transport["command"].(string)
			if cmd == "" {
				r.add(SeverityError, prefix+".transport.command", "command is required for stdio transport")
			} else {
				r.add(SeverityInfo, prefix+".transport.command", fmt.Sprintf("✓ %s", cmd))
			}
		case "":
			r.add(SeverityError, prefix+".transport.type", "transport type is required — use \"http\" or \"stdio\"")
		default:
			r.add(SeverityWarning, prefix+".transport.type",
				fmt.Sprintf("%q is not a known transport type — use \"http\" or \"stdio\"", tType))
		}

		// authentication
		if auth, ok := tool["authentication"].(map[string]interface{}); ok {
			authType, _ := auth["type"].(string)
			switch authType {
			case "api-key":
				key, _ := auth["api_key"].(string)
				if key == "" {
					r.add(SeverityError, prefix+".authentication.api_key",
						"api_key is empty — use \"${env:VAR_NAME}\"")
				} else if !reEnvPlaceholder.MatchString(key) {
					r.add(SeverityWarning, prefix+".authentication.api_key",
						"looks like a hardcoded key — use \"${env:VAR_NAME}\" instead")
				}
			case "bearer":
				token, _ := auth["token"].(string)
				if token == "" {
					r.add(SeverityError, prefix+".authentication.token", "token is empty for bearer auth")
				}
			case "basic":
				user, _ := auth["username"].(string)
				pass, _ := auth["password"].(string)
				if user == "" {
					r.add(SeverityError, prefix+".authentication.username", "username is required for basic auth")
				}
				if pass == "" {
					r.add(SeverityError, prefix+".authentication.password", "password is required for basic auth")
				}
			}
		}
	}
}

// Interfaces 

func checkInterfaces(r *Report, s *parser.Spec) {
	if len(s.Interfaces) == 0 {
		r.add(SeverityInfo, "interfaces", "not set — agent will default to consolechat")
		return
	}

	validTypes := map[string]bool{
		"webchat": true, "consolechat": true, "webhook": true,
	}

	for i, iface := range s.Interfaces {
		prefix := fmt.Sprintf("interfaces[%d]", i)
		if iface.Type == "" {
			r.add(SeverityError, prefix+".type", "interface type is required")
		} else if !validTypes[iface.Type] {
			r.add(SeverityWarning, prefix+".type",
				fmt.Sprintf("%q is not a known interface type — use webchat, consolechat, or webhook", iface.Type))
		} else {
			r.add(SeverityInfo, prefix+".type", fmt.Sprintf("✓ %s", iface.Type))
		}

		// webhook-specific checks
		if iface.Type == "webhook" {
			if iface.Prompt == "" {
				r.add(SeverityWarning, prefix+".prompt",
					"webhook interface should have a prompt template with ${http:payload.*} placeholders")
			}
		}
	}
}

// Skills 

func checkSkills(r *Report, s *parser.Spec) {
	for i, skill := range s.Skills {
		prefix := fmt.Sprintf("skills[%d]", i)
		switch skill.Type {
		case "local":
			if skill.Path == "" {
				r.add(SeverityError, prefix+".path", "path is required for local skills")
			} else {
				r.add(SeverityInfo, prefix+".path", fmt.Sprintf("✓ %s", skill.Path))
			}
		case "remote":
			if skill.URL == "" {
				r.add(SeverityError, prefix+".url", "url is required for remote skills")
			} else {
				r.add(SeverityInfo, prefix+".url", fmt.Sprintf("✓ %s", skill.URL))
			}
		case "":
			r.add(SeverityError, prefix+".type", "skill type is required — use \"local\" or \"remote\"")
		default:
			r.add(SeverityWarning, prefix+".type",
				fmt.Sprintf("%q is not a known skill type — use \"local\" or \"remote\"", skill.Type))
		}
	}
}

// Output Schema 

func checkOutputSchema(r *Report, s *parser.Spec) {
	if s.OutputSchema == "" {
		return // optional
	}
	// Validate it's parseable JSON
	var js interface{}
	if err := json.Unmarshal([]byte(s.OutputSchema), &js); err != nil {
		r.add(SeverityError, "# Output Schema",
			fmt.Sprintf("JSON in output schema is not valid: %v", err))
	} else {
		r.add(SeverityInfo, "# Output Schema", "✓ valid JSON")
	}
}

// helpers

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
