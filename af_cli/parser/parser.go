// Package parser reads and parses AgentFactory .md spec files.
// Handles YAML frontmatter + markdown sections (Role, Instructions, etc.)
package parser

import (
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

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

// FieldIndex maps a field path string to its source position.
type FieldIndex map[string]Position

// Position holds a 1-based line and column number.
type Position struct {
	Line int
	Col  int
}

// TypoHint describes a misspelled key found in the source file.
type TypoHint struct {
	Line       int
	Col        int
	Found      string // the key as written in the file
	Suggestion string // the correct key name
	Context    string // e.g. "model", "interfaces[0]", "root"
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
	// Normalize line endings
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Extract YAML frontmatter
	fm := reFrontmatter.FindStringSubmatch(content)
	if fm == nil {
		return nil, fmt.Errorf("no YAML frontmatter found — file must start with ---")
	}

	spec := &Spec{}
	if err := yaml.Unmarshal([]byte(fm[1]), spec); err != nil {
		return nil, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	// Build position index from the raw source
	spec.Positions, spec.TypoHints = buildIndex(content)

	// Parse markdown body sections
	body := content[len(fm[0]):]
	segments := reSeparator.Split(body, -1)

	for _, seg := range segments {
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
			cb := reCodeBlock.FindStringSubmatch(bodyText)
			if cb != nil {
				spec.OutputSchema = strings.TrimSpace(cb[1])
			}
		}
	}

	return spec, nil
}

// validKeysForContext maps a parent context path to the set of valid child keys.
// "root" covers top-level frontmatter keys.
var validKeysForContext = map[string][]string{
	"root": {
		"spec_version", "name", "description", "version", "license",
		"author", "max_iterations", "execution_mode", "model",
		"interfaces", "tools", "skills", "memory",
	},
	"model": {
		"provider", "name", "temperature", "base_url", "authentication",
	},
	"model.authentication": {
		"type", "api_key", "token", "username", "password",
	},
	"interfaces[n]": {
		"type", "prompt", "exposure", "subscription",
	},
	"tools.mcp[n]": {
		"name", "transport", "authentication", "env", "tool_filter", "query_params",
	},
	"tools.mcp[n].transport": {
		"type", "url", "command", "args",
	},
	"tools.mcp[n].authentication": {
		"type", "api_key", "token", "username", "password",
	},
	"tools.mcp[n].query_params[n]": {
		"key", "description", "required", "default",
	},
	"skills[n]": {
		"type", "path", "url",
	},
	"memory": {
		"type",
	},
}

// contextPattern maps a path pattern to a context key in validKeysForContext.
// Patterns with [n] match any indexed list item.
func contextForPath(parentPath string) ([]string, string) {
	// Exact match first
	if keys, ok := validKeysForContext[parentPath]; ok {
		return keys, parentPath
	}
	// Normalize indexed paths: replace [0], [1], ... with [n]
	normalized := reIndex.ReplaceAllString(parentPath, "[n]")
	if keys, ok := validKeysForContext[normalized]; ok {
		return keys, normalized
	}
	// Root level
	if parentPath == "" {
		return validKeysForContext["root"], "root"
	}
	return nil, ""
}

var reIndex = regexp.MustCompile(`\[\d+\]`)

// levenshtein computes the edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
		dp[i][0] = i
	}
	for j := 1; j <= lb; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			dp[i][j] = min3(dp[i-1][j]+1, dp[i][j-1]+1, dp[i-1][j-1]+cost)
		}
	}
	return dp[la][lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// suggestKey returns the closest valid key if the given key looks like a typo,
// or empty string if no close match is found.
func suggestKey(key string, validKeys []string) string {
	best := ""
	bestDist := 3 // max edit distance to consider a typo (≤ 2)
	for _, valid := range validKeys {
		d := levenshtein(key, valid)
		if d < bestDist {
			bestDist = d
			best = valid
		}
	}
	return best
}

// buildIndex scans the raw file content line by line and records the position
// of every key we care about in the validator.
func buildIndex(content string) (FieldIndex, []TypoHint) {
	idx := make(FieldIndex)
	var typos []TypoHint
	lines := strings.Split(content, "\n")

	type frame struct {
		indent  int
		key     string
		listIdx int
	}
	stack := []frame{}
	listCounters := map[string]int{}

	inFrontmatter := false
	frontmatterClosed := false
	dashCount := 0

	checkTypo := func(key, parentPath string, line, col int) {
		validKeys, ctx := contextForPath(parentPath)
		if validKeys == nil {
			return
		}
		// Key is already valid — no typo
		for _, v := range validKeys {
			if key == v {
				return
			}
		}
		if suggestion := suggestKey(key, validKeys); suggestion != "" {
			typos = append(typos, TypoHint{
				Line:       line,
				Col:        col,
				Found:      key,
				Suggestion: suggestion,
				Context:    ctx,
			})
		}
	}

	for lineNum, raw := range lines {
		line := lineNum + 1
		trimmed := strings.TrimRight(raw, " \t")

		if strings.TrimSpace(trimmed) == "---" {
			dashCount++
			if dashCount == 1 {
				inFrontmatter = true
			} else if dashCount == 2 {
				inFrontmatter = false
				frontmatterClosed = true
			}
			continue
		}

		if frontmatterClosed {
			stripped := strings.TrimSpace(trimmed)
			if strings.HasPrefix(stripped, "# ") {
				heading := strings.ToLower(strings.TrimPrefix(stripped, "# "))
				switch heading {
				case "role":
					idx["# Role"] = Position{Line: line, Col: 1}
				case "instructions":
					idx["# Instructions"] = Position{Line: line, Col: 1}
				case "enforcement":
					idx["# Enforcement"] = Position{Line: line, Col: 1}
				case "output schema":
					idx["# Output Schema"] = Position{Line: line, Col: 1}
				}
			}
			continue
		}

		if !inFrontmatter {
			continue
		}

		if trimmed == "" || strings.HasPrefix(strings.TrimSpace(trimmed), "#") {
			continue
		}

		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		stripped := strings.TrimSpace(trimmed)

		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}

		parentPath := ""
		if len(stack) > 0 {
			parentPath = stack[len(stack)-1].key
		}

		if strings.HasPrefix(stripped, "- ") {
			rest := strings.TrimPrefix(stripped, "- ")
			col := indent + 1

			counterKey := parentPath
			n := listCounters[counterKey]
			listCounters[counterKey] = n + 1

			itemPath := fmt.Sprintf("%s[%d]", parentPath, n)
			if parentPath == "" {
				itemPath = fmt.Sprintf("[%d]", n)
			}

			colonIdx := strings.Index(rest, ":")
			if colonIdx >= 0 {
				key := strings.TrimSpace(rest[:colonIdx])
				fieldPath := itemPath + "." + key
				idx[fieldPath] = Position{Line: line, Col: col}
				registerAliases(idx, fieldPath, line, col)
				checkTypo(key, reIndex.ReplaceAllString(itemPath, "[n]"), line, col)
			}

			idx[itemPath] = Position{Line: line, Col: col}
			stack = append(stack, frame{indent: indent, key: itemPath})
			continue
		}

		colonIdx := strings.Index(stripped, ":")
		if colonIdx < 0 {
			continue
		}

		key := strings.TrimSpace(stripped[:colonIdx])
		col := indent + 1

		path := key
		if parentPath != "" {
			path = parentPath + "." + key
		}

		idx[path] = Position{Line: line, Col: col}
		registerAliases(idx, path, line, col)

		// Determine the context for typo checking
		normalizedParent := reIndex.ReplaceAllString(parentPath, "[n]")
		checkTypo(key, normalizedParent, line, col)

		stack = append(stack, frame{indent: indent, key: path})
	}

	return idx, typos
}

// registerAliases maps the dotted YAML path to the field names used in
// validator emit() calls so lookups work without exact path matching.
func registerAliases(idx FieldIndex, path string, line, col int) {
	pos := Position{Line: line, Col: col}

	// Static top-level aliases
	staticAliases := map[string]string{
		"spec_version":                 "spec_version",
		"name":                         "name",
		"version":                      "version",
		"description":                  "description",
		"max_iterations":               "max_iterations",
		"execution_mode":               "execution_mode",
		"model.provider":               "model.provider",
		"model.name":                   "model.name",
		"model.temperature":            "model.temperature",
		"model.authentication":         "model.authentication",
		"model.authentication.type":    "model.authentication.type",
		"model.authentication.api_key": "model.authentication.api_key",
		"model.authentication.token":   "model.authentication.token",
		"interfaces":                   "interfaces",
		"tools":                        "tools",
		"skills":                       "skills",
	}

	if alias, ok := staticAliases[path]; ok {
		if _, exists := idx[alias]; !exists {
			idx[alias] = pos
		}
	}

	// Dynamic: interfaces[n].type  →  interfaces[n].type  (already the right key)
	// Dynamic: tools.mcp[n].name   →  tools[n].name  (validator uses tools.NAME.name)
	// These are already stored with the correct indexed key by buildIndex,
	// so no extra aliasing needed — the validator field strings match directly.
}
