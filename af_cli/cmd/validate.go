// Package cmd implements CLI subcommands.
package cmd

import (
	"fmt"
	"os"
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

	fmt.Printf("%s%s%s%s (%d)\n", colorBold, color, label, colorReset, len(items))
	for _, r := range items {
		fmt.Printf("  %s[%s]%s %-30s %s\n",
			colorDim, r.RuleID, colorReset, r.Field, r.Message)
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
