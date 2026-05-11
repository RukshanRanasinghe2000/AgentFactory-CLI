// Package parser reads and parses AgentFactory .md spec files.
// Handles YAML frontmatter + markdown sections (Role, Instructions, etc.)
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
type FieldIndex = lexer.FieldIndex
type Position = lexer.Position
type TypoHint = lexer.TypoHint

// Spec holds the parsed contents of an agent .md file.
type Spec struct {
	// Frontmatter fields
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

	// Markdown section fields (populated after parsing body)
	Role         string
	Instructions string
	Enforcement  string
	OutputSchema string // raw JSON from ```json block under # Output Schema

	// Position index: maps a field key to its (line, col) in the source file.
	Positions FieldIndex

	// TypoHints holds detected misspelled keys with suggested corrections.
	TypoHints []TypoHint
}

type ModelSpec struct {
	Provider       string    `yaml:"provider"`
	Name           string    `yaml:"name"`
	Temperature    float64   `yaml:"temperature"`
	BaseURL        string    `yaml:"base_url"`
	Authentication *AuthSpec `yaml:"authentication"`
}

type AuthSpec struct {
	Type     string `yaml:"type"`
	APIKey   string `yaml:"api_key"`
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Interface struct {
	Type         string                 `yaml:"type"`
	Prompt       string                 `yaml:"prompt"`
	Exposure     map[string]interface{} `yaml:"exposure"`
	Subscription map[string]string      `yaml:"subscription"`
}

type Skill struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
	URL  string `yaml:"url"`
}

type MemorySpec struct {
	Type string `yaml:"type"`
}

type ToolMCP struct {
	Name      string        `yaml:"name"`
	Transport ToolTransport `yaml:"transport"`
	Auth      *AuthSpec     `yaml:"authentication"`
}

type ToolTransport struct {
	Type    string   `yaml:"type"`
	URL     string   `yaml:"url"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

var (
	reFrontmatter = regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n`)
	reSeparator   = regexp.MustCompile(`\n---+\n`)
	reHeading     = regexp.MustCompile(`(?s)^#\s+(.+?)\n(.*)`)
	reCodeBlock   = regexp.MustCompile("(?s)```(?:json)?\\n([\\s\\S]*?)```")
)

// ParseFile reads a .md spec file and returns a parsed Spec with position info.
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

	// Delegate scanning to the lexer
	result := lexer.Scan(content)
	spec.Positions = result.Positions
	spec.TypoHints = result.Typos

	// Parse markdown body sections
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
