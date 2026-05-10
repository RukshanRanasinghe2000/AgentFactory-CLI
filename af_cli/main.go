package main

import (
	"fmt"
	"os"

	"github.com/agentfactory/cli/cmd"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		file, configPath := parseValidateArgs(os.Args[2:])
		if file == "" {
			fmt.Fprintln(os.Stderr, "Usage: agentfactory validate <agent.md> [--config <path>]")
			os.Exit(1)
		}
		cmd.Validate(file, configPath)

	case "version", "--version", "-v":
		fmt.Println("agentfactory-cli v0.1.0")

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// parseValidateArgs extracts the spec file path and optional --config value
// from the args slice after "validate".
func parseValidateArgs(args []string) (file, configPath string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config", "-c":
			if i+1 < len(args) {
				configPath = args[i+1]
				i++
			}
		default:
			if file == "" {
				file = args[i]
			}
		}
	}
	return
}

func printUsage() {
	fmt.Println(`AgentFactory CLI — validate .md agent specs

Usage:
  agentfactory validate <agent.md>                    Validate using auto-discovered config
  agentfactory validate <agent.md> --config <path>    Validate using a specific config file
  agentfactory version                                Show CLI version
  agentfactory help                                   Show this help

Config file (.afvalidate.toml) search order:
  1. --config flag path
  2. Same directory as the spec file
  3. Current working directory
  4. Home directory (~/.afvalidate.toml)

Examples:
  agentfactory validate weather.md
  agentfactory validate agents/code-reviewer.md --config team.afvalidate.toml`)
}
