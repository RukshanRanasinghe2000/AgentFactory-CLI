// Package config loads and resolves .afvalidate.toml rule configuration.
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// RuleLevel controls how a rule behaves.
// Valid values: "error", "warn", "info", "off"
type RuleLevel string

const (
	LevelError   RuleLevel = "error"
	LevelWarn    RuleLevel = "warn"
	LevelInfo    RuleLevel = "info"
	LevelOff     RuleLevel = "off"
)

// Config is the parsed contents of .afvalidate.toml.
type Config struct {
	// Rules maps rule ID → level override ("error", "warn", "info", "off")
	Rules map[string]RuleLevel `toml:"rules"`
}

// DefaultConfig returns a Config with no overrides (all rules use built-in defaults).
func DefaultConfig() *Config {
	return &Config{Rules: make(map[string]RuleLevel)}
}

// Load finds and parses the nearest .afvalidate.toml.
// Search order:
//  1. explicit path (if non-empty)
//  2. same directory as the spec file
//  3. current working directory
//  4. user home directory (~/.afvalidate.toml)
//
// Returns DefaultConfig if no file is found.
func Load(explicitPath, specFile string) (*Config, string, error) {
	candidates := buildCandidates(explicitPath, specFile)

	for _, path := range candidates {
		cfg, err := parseFile(path)
		if err != nil {
			return nil, "", err
		}
		if cfg != nil {
			return cfg, path, nil
		}
	}

	return DefaultConfig(), "", nil
}

func buildCandidates(explicitPath, specFile string) []string {
	var paths []string

	if explicitPath != "" {
		paths = append(paths, explicitPath)
		return paths // explicit path is authoritative — don't fall through
	}

	const filename = ".afvalidate.toml"

	// Directory of the spec file
	if specFile != "" {
		paths = append(paths, filepath.Join(filepath.Dir(specFile), filename))
	}

	// Current working directory
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, filename))
	}

	// Home directory
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, filename))
	}

	return paths
}

func parseFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // not found — try next candidate
		}
		return nil, err
	}

	cfg := &Config{Rules: make(map[string]RuleLevel)}
	if _, err := toml.Decode(string(data), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Resolve returns the effective level for a rule ID.
// If the rule is overridden in config, that level is returned.
// Otherwise the built-in default is returned.
func (c *Config) Resolve(ruleID string, defaultLevel RuleLevel) RuleLevel {
	if c == nil {
		return defaultLevel
	}
	if level, ok := c.Rules[ruleID]; ok {
		return normalize(level)
	}
	return defaultLevel
}

// IsOff returns true if the rule is disabled in config.
func (c *Config) IsOff(ruleID string) bool {
	return c.Resolve(ruleID, "") == LevelOff
}

func normalize(l RuleLevel) RuleLevel {
	switch RuleLevel(strings.ToLower(string(l))) {
	case LevelError:
		return LevelError
	case LevelWarn, "warning":
		return LevelWarn
	case LevelInfo:
		return LevelInfo
	case LevelOff, "disable", "disabled", "false":
		return LevelOff
	default:
		return l
	}
}
