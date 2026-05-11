// Package cmd implements CLI subcommands.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentfactory/cli/config"
	"github.com/agentfactory/cli/parser"
	"github.com/agentfactory/cli/validator"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// Validate reads, parses, and validates a .md agent spec file.
// Accepts an optional explicit config path (empty string = auto-discover).
// Exits with code 1 if any errors are found.
func Validate(file, configPath string) {
	fmt.Println(configPath)
	// Load config (.afvalidate.toml)
	cfg, cfgPath, err := config.Load(configPath, file)
	if err != nil {
		fatalf("config error: %v", err)
	}

	// Read spec file
	data, err := os.ReadFile(file)
	if err != nil {
		fatalf("cannot read file: %v", err)
	}

	// Parse
	spec, err := parser.ParseFile(string(data))
	if err != nil {
		fatalf("parse error: %v", err)
	}

	// Validate
	report := validator.Validate(file, spec, cfg)

	// Print header
	fmt.Printf("\n%s%sAgentFactory Spec Validator%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("%s%s%s\n", colorDim, file, colorReset)
	if cfgPath != "" {
		fmt.Printf("%sconfig: %s%s\n", colorDim, cfgPath, colorReset)
	}
	fmt.Println()

	// Print results grouped by severity
	printGroup(report, validator.SeverityError, colorRed, "✗ Errors")
	printGroup(report, validator.SeverityWarning, colorYellow, "⚠ Warnings")
	printGroup(report, validator.SeverityInfo, colorGreen, "✓ Passed")

	// Summary line
	fmt.Println(strings.Repeat("─", 60))
	printSummary(report)
	fmt.Println()

	if report.Errors > 0 {
		os.Exit(1)
	}
}

// ValidateDir finds every .md file in dir, validates each one, and prints
// a combined summary. Exits with code 1 if any file has errors.
func ValidateDir(dir, configPath string) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil || len(entries) == 0 {
		fatalf("no .md files found in directory: %s", dir)
	}

	fmt.Printf("\n%s%sAgentFactory Spec Validator%s  %s(directory)%s\n",
		colorBold, colorCyan, colorReset, colorDim, colorReset)
	fmt.Printf("%s%s%s\n\n", colorDim, dir, colorReset)

	totalFiles := 0
	totalErrors := 0
	totalWarnings := 0

	for _, file := range entries {
		cfg, _, err := config.Load(configPath, file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%serror loading config for %s: %v%s\n",
				colorRed, file, err, colorReset)
			continue
		}

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%scannot read %s: %v%s\n",
				colorRed, file, err, colorReset)
			continue
		}

		spec, err := parser.ParseFile(string(data))
		if err != nil {
			// Not a valid spec — skip silently with a note
			fmt.Printf("%s%-40s%s %sskipped (not a valid spec)%s\n",
				colorDim, filepath.Base(file), colorReset, colorYellow, colorReset)
			continue
		}

		report := validator.Validate(file, spec, cfg)
		totalFiles++
		totalErrors += report.Errors
		totalWarnings += report.Warnings

		// One-line result per file
		base := filepath.Base(file)
		if report.Errors > 0 {
			fmt.Printf("  %s✗%s %-40s %s%d error(s)%s",
				colorRed, colorReset, base, colorRed, report.Errors, colorReset)
		} else {
			fmt.Printf("  %s✓%s %-40s %s%d passed%s",
				colorGreen, colorReset, base, colorGreen, report.Infos, colorReset)
		}
		if report.Warnings > 0 {
			fmt.Printf("  %s%d warning(s)%s", colorYellow, report.Warnings, colorReset)
		}
		fmt.Println()
	}

	// Overall summary
	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("%s files: %d  |  ", colorBold, totalFiles)
	if totalErrors > 0 {
		fmt.Printf("%s%d error(s)%s  |  ", colorRed, totalErrors, colorReset+colorBold)
	}
	if totalWarnings > 0 {
		fmt.Printf("%s%d warning(s)%s  |  ", colorYellow, totalWarnings, colorReset+colorBold)
	}
	fmt.Printf("%s\n\n", colorReset)

	if totalErrors > 0 {
		os.Exit(1)
	}
}

func printGroup(report *validator.Report, sev validator.Severity, color, label string) {
	var items []validator.Result
	for _, r := range report.Results {
		if r.Severity == sev {
			items = append(items, r)
		}
	}
	if len(items) == 0 {
		return
	}

	// For info severity, split into genuine passes (✓ prefix) and notices.
	if sev == validator.SeverityInfo {
		var passes, notices []validator.Result
		for _, r := range items {
			if strings.HasPrefix(r.Message, "✓") {
				passes = append(passes, r)
			} else {
				notices = append(notices, r)
			}
		}
		if len(notices) > 0 {
			fmt.Printf("%s%sℹ Notices%s (%d)\n", colorBold, colorCyan, colorReset, len(notices))
			for _, r := range notices {
				loc := ""
				if r.Line > 0 {
					loc = fmt.Sprintf("%s:%d:%d%s", colorDim, r.Line, r.Col, colorReset)
				}
				fmt.Printf("  %s[%s]%s %-30s %-10s %s\n",
					colorDim, r.RuleID, colorReset, r.Field, loc, r.Message)
			}
			fmt.Println()
		}
		if len(passes) > 0 {
			fmt.Printf("%s%s%s%s (%d)\n", colorBold, color, label, colorReset, len(passes))
			for _, r := range passes {
				loc := ""
				if r.Line > 0 {
					loc = fmt.Sprintf("%s:%d:%d%s", colorDim, r.Line, r.Col, colorReset)
				}
				fmt.Printf("  %s[%s]%s %-30s %-10s %s\n",
					colorDim, r.RuleID, colorReset, r.Field, loc, r.Message)
			}
			fmt.Println()
		}
		return
	}

	fmt.Printf("%s%s%s%s (%d)\n", colorBold, color, label, colorReset, len(items))
	for _, r := range items {
		loc := ""
		if r.Line > 0 {
			loc = fmt.Sprintf("%s:%d:%d%s", colorDim, r.Line, r.Col, colorReset)
		}
		fmt.Printf("  %s[%s]%s %-30s %-10s %s\n",
			colorDim, r.RuleID, colorReset, r.Field, loc, r.Message)
	}
	fmt.Println()
}

func printSummary(report *validator.Report) {
	parts := []string{}
	if report.Errors > 0 {
		parts = append(parts, fmt.Sprintf("%s%s%d error(s)%s", colorBold, colorRed, report.Errors, colorReset))
	}
	if report.Warnings > 0 {
		parts = append(parts, fmt.Sprintf("%s%s%d warning(s)%s", colorBold, colorYellow, report.Warnings, colorReset))
	}
	if report.Infos > 0 {
		parts = append(parts, fmt.Sprintf("%s%s%d passed%s", colorBold, colorGreen, report.Infos, colorReset))
	}

	if report.Errors == 0 {
		fmt.Printf("%s%s✓ Spec is valid%s  —  %s\n",
			colorBold, colorGreen, colorReset, strings.Join(parts, "  "))
	} else {
		fmt.Printf("%s%s✗ Spec has errors%s  —  %s\n",
			colorBold, colorRed, colorReset, strings.Join(parts, "  "))
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, colorRed+"error: "+colorReset+format+"\n", args...)
	os.Exit(1)
}
