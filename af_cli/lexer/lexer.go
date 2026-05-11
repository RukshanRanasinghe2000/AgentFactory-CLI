// Package lexer scans raw AgentFactory .md source files line by line.
// It builds a position index (field path → line/col) and detects
// misspelled YAML keys using Levenshtein distance matching.
package lexer

import (
	"fmt"
	"regexp"
	"strings"
)

// FieldIndex maps a dotted field path to its source position.
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
	Context    string // e.g. "model", "interfaces[n]", "root"
}

// ScanResult is the output of Scan.
type ScanResult struct {
	Positions FieldIndex
	Typos     []TypoHint
}

// Scan walks the raw .md file content and returns a position index and any
// typo hints found in the YAML frontmatter and markdown section headings.
func Scan(content string) ScanResult {
	idx, typos := buildIndex(content)
	return ScanResult{Positions: idx, Typos: typos}
}

// Valid key registry 

// validKeysForContext maps a parent context path to the set of valid child keys.
// Indexed paths use [n] as a wildcard (e.g. "interfaces[n]").
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

var reIndex = regexp.MustCompile(`\[\d+\]`)

// contextForPath returns the valid keys and context label for a given parent path.
// Indexed segments like [0] are normalized to [n] before lookup.
func contextForPath(parentPath string) ([]string, string) {
	if keys, ok := validKeysForContext[parentPath]; ok {
		return keys, parentPath
	}
	normalized := reIndex.ReplaceAllString(parentPath, "[n]")
	if keys, ok := validKeysForContext[normalized]; ok {
		return keys, normalized
	}
	if parentPath == "" {
		return validKeysForContext["root"], "root"
	}
	return nil, ""
}

// Levenshtein / typo matching 

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

// suggestKey returns the closest valid key within edit distance 2,
// or empty string if no close match is found.
func suggestKey(key string, validKeys []string) string {
	best := ""
	bestDist := 3 // threshold: only suggest if distance < 3
	for _, valid := range validKeys {
		if d := levenshtein(key, valid); d < bestDist {
			bestDist = d
			best = valid
		}
	}
	return best
}

// Index builder 

// buildIndex scans the raw file content line by line, records the source
// position of every YAML key, and detects misspelled keys.
func buildIndex(content string) (FieldIndex, []TypoHint) {
	idx := make(FieldIndex)
	var typos []TypoHint
	lines := strings.Split(content, "\n")

	type frame struct {
		indent int
		key    string // full dotted path of this frame
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
		for _, v := range validKeys {
			if key == v {
				return // valid key — no typo
			}
		}
		if suggestion := suggestKey(key, validKeys); suggestion != "" {
			typos = append(typos, TypoHint{
				Line: line, Col: col,
				Found: key, Suggestion: suggestion, Context: ctx,
			})
		}
	}

	for lineNum, raw := range lines {
		line := lineNum + 1
		trimmed := strings.TrimRight(raw, " \t")

		// Frontmatter boundary detection
		if strings.TrimSpace(trimmed) == "---" {
			dashCount++
			switch dashCount {
			case 1:
				inFrontmatter = true
			case 2:
				inFrontmatter = false
				frontmatterClosed = true
			}
			continue
		}

		// Markdown section headings (after frontmatter closes)
		if frontmatterClosed {
			if stripped := strings.TrimSpace(trimmed); strings.HasPrefix(stripped, "# ") {
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

		// Skip blank lines and YAML comments
		if trimmed == "" || strings.HasPrefix(strings.TrimSpace(trimmed), "#") {
			continue
		}

		indent := len(raw) - len(strings.TrimLeft(raw, " \t"))
		stripped := strings.TrimSpace(trimmed)

		// Pop stack frames at same or deeper indent
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}

		// Current parent path is the top of the stack
		parentPath := ""
		if len(stack) > 0 {
			parentPath = stack[len(stack)-1].key
		}

		// List item: "- key: value" 
		if strings.HasPrefix(stripped, "- ") {
			rest := strings.TrimPrefix(stripped, "- ")
			col := indent + 1

			n := listCounters[parentPath]
			listCounters[parentPath] = n + 1

			itemPath := fmt.Sprintf("%s[%d]", parentPath, n)
			if parentPath == "" {
				itemPath = fmt.Sprintf("[%d]", n)
			}

			if colonIdx := strings.Index(rest, ":"); colonIdx >= 0 {
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

		// Regular key: "key: value" or "key:" 
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
		checkTypo(key, reIndex.ReplaceAllString(parentPath, "[n]"), line, col)

		stack = append(stack, frame{indent: indent, key: path})
	}

	return idx, typos
}

// registerAliases records shorthand field names used by the validator
// alongside the full dotted path so position lookups work for both forms.
func registerAliases(idx FieldIndex, path string, line, col int) {
	pos := Position{Line: line, Col: col}

	aliases := map[string]string{
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

	if alias, ok := aliases[path]; ok {
		if _, exists := idx[alias]; !exists {
			idx[alias] = pos
		}
	}
}
