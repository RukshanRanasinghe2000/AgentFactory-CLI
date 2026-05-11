// Package parser reads and parses AgentFactory .md spec files.
// It handles YAML frontmatter unmarshalling and markdown section extraction.
// Position indexing and typo detection are delegated to the lexer package.
package parser

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/agentfactory/cli/lexer"
	"gopkg.in/yaml.v3"
)

// Re-export lexer types so callers only need to import parser.
// FieldIndex, Position, and TypoHint are defined in the lexer package.
type FieldIndex = lexer.FieldIndex
type Position = lexer.Position
type TypoHint = lexer.TypoHint

// Spec holds the fully parsed contents of an AgentFactory .md file.
// Frontmatter fields are populated by YAML unmarshalling; markdown section
// fields (Role, Instructions, Enforcement, OutputSchema) are extracted from
// the body. Positions and TypoHints are produced by the lexer.
type Spec struct {
	// ── Frontmatter ──────────────────────────────────────────────────────────
	SpecVersion   string      `yaml:"spec_version"`
	Name          string      `yaml:"name"`
	Description   string      `yaml:"description"`
	Version       string      `yaml:"version"`
	License       string      `yaml:"license"`
	Author        string      `yaml:"author"`
	MaxIterations int         `yaml:"max_iterations"`
	ExecutionMode string      `yaml:"execution_mode"`
	Model         ModelSpec   `yaml:"model"`
	Interfaces    []Interface `yaml:"interfaces"`
	Tools         interface{} `yaml:"tools"`
	Skills        []Skill     `yaml:"skills"`
	Memory        MemorySpec  `yaml:"memory"`

	// ── Markdown body sections ────────────────────────────────────────────────
	Role         string
	Instructions string
	Enforcement  string
	OutputSchema string // raw JSON extracted from the ```json block under # Output Schema

	// ── Lexer output ─────────────────────────────────────────────────────────
	// Positions maps every known field path to its (line, col) in the source.
	Positions FieldIndex
	// TypoHints holds misspelled keys detected by the lexer.
	TypoHints []TypoHint
}

// ModelSpec holds the model configuration block from the frontmatter.
type ModelSpec struct {
	Provider       string    `yaml:"provider"`
	Name           string    `yaml:"name"`
	Temperature    float64   `yaml:"temperature"`
	BaseURL        string    `yaml:"base_url"`
	Authentication *AuthSpec `yaml:"authentication"`
}

// AuthSpec holds authentication credentials for a model or tool.
// The Type field selects the auth scheme: "api-key", "bearer", or "basic".
type AuthSpec struct {
	Type     string `yaml:"type"`
	APIKey   string `yaml:"api_key"`
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Interface describes a single entry in the interfaces list.
// Type must be one of "webchat", "consolechat", or "webhook".
type Interface struct {
	Type         string                 `yaml:"type"`
	Prompt       string                 `yaml:"prompt"`
	Exposure     map[string]interface{} `yaml:"exposure"`
	Subscription map[string]string      `yaml:"subscription"`
}

// Skill describes a local or remote skill attached to the agent.
type Skill struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
	URL  string `yaml:"url"`
}

// MemorySpec holds the memory configuration block.
type MemorySpec struct {
	Type string `yaml:"type"`
}

// ToolMCP describes a single MCP tool entry inside the tools.mcp list.
type ToolMCP struct {
	Name      string        `yaml:"name"`
	Transport ToolTransport `yaml:"transport"`
	Auth      *AuthSpec     `yaml:"authentication"`
}

// ToolTransport describes how the agent connects to an MCP tool.
// Type is "http" (URL required) or "stdio" (Command + Args required).
type ToolTransport struct {
	Type    string   `yaml:"type"`
	URL     string   `yaml:"url"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

// Compiled regexes used during markdown body parsing.
var (
	// reFrontmatter matches the opening --- ... --- block at the start of the file.
	reFrontmatter = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n`)
	// reSeparator splits the markdown body on --- dividers between sections.
	reSeparator = regexp.MustCompile(`\n---+\n`)
	// reHeading matches a top-level "# Heading\n body" pattern.
	reHeading = regexp.MustCompile(`(?s)^#\s+(.+?)\n(.*)`)
	// reCodeBlock extracts the content of a fenced ```json ... ``` block.
	reCodeBlock = regexp.MustCompile("(?s)```(?:json)?\\n([\\s\\S]*?)```")
)

// ParseFile parses the raw content of an AgentFactory .md spec file and
// returns a populated Spec. It normalises line endings, extracts and
// unmarshals the YAML frontmatter, delegates source scanning to the lexer,
// and then walks the markdown body to populate Role, Instructions,
// Enforcement, and OutputSchema. Returns an error if the frontmatter is
// missing or contains invalid YAML.
func ParseFile(content string) (*Spec, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")

	fm := reFrontmatter.FindStringSubmatch(content)
	if fm == nil {
		return nil, fmt.Errorf("no YAML frontmatter found — file must start with ---")
	}

	spec := &Spec{}
	if err := yaml.Unmarshal([]byte(fm[1]), spec); err != nil {
		return nil, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	// Delegate position indexing and typo detection to the lexer.
	result := lexer.Scan(content)
	spec.Positions = result.Positions
	spec.TypoHints = result.Typos

	// Walk the markdown body and populate section fields.
	body := content[len(fm[0]):]
	for _, seg := range reSeparator.Split(body, -1) {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		m := reHeading.FindStringSubmatch(seg)
		if m == nil {
			continue
		}
		title := strings.ToLower(strings.TrimSpace(m[1]))
		bodyText := strings.TrimSpace(m[2])

		switch title {
		case "role":
			spec.Role = bodyText
		case "instructions":
			spec.Instructions = bodyText
		case "enforcement":
			spec.Enforcement = bodyText
		case "output schema":
			if cb := reCodeBlock.FindStringSubmatch(bodyText); cb != nil {
				spec.OutputSchema = strings.TrimSpace(cb[1])
			}
		}
	}

	return spec, nil
}
