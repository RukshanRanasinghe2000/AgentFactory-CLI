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
		file, dir, configPath := parseValidateArgs(os.Args[2:])

		if dir != "" {
			cmd.ValidateDir(dir, configPath)
		} else if file != "" {
			cmd.Validate(file, configPath)
		} else {
			fmt.Fprintln(os.Stderr, "Usage: agentfactory validate <agent.md> [--config <path>]")
			fmt.Fprintln(os.Stderr, "       agentfactory validate -d <directory> [--config <path>]")
			os.Exit(1)
		}

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

// parseValidateArgs extracts the spec file, optional directory (-d), and
// optional --config value from the args slice after "validate".
func parseValidateArgs(args []string) (file, dir, configPath string) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-d":
			if i+1 < len(args) {
				dir = args[i+1]
				i++
			} else {
				fmt.Fprintln(os.Stderr, "error: -d requires a directory path")
				os.Exit(1)
			}
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
  agentfactory validate <agent.md>                      Validate a single spec file
  agentfactory validate -d <directory>                  Validate all .md files in a directory
  agentfactory validate <agent.md> --config <path>      Use a specific config file
  agentfactory validate -d <directory> --config <path>  Directory validate with config
  agentfactory version                                  Show CLI version
  agentfactory help                                     Show this help

Config file (.afvalidate.toml) search order:
  1. --config flag path
  2. Same directory as the spec file
  3. Current working directory
  4. Home directory (~/.afvalidate.toml)

Examples:
  agentfactory validate weather.md
  agentfactory validate -d ./agents
  agentfactory validate -d ./agents --config team.afvalidate.toml`)
}
