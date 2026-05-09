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
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: agentfactory validate <agent.md>")
			os.Exit(1)
		}
		cmd.Validate(os.Args[2])
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

func printUsage() {
	fmt.Println(`AgentFactory CLI — validate and run .md agent specs

Usage:
  agentfactory validate <agent.md>   Validate an agent spec file
  agentfactory version               Show CLI version
  agentfactory help                  Show this help

Examples:
  agentfactory validate weather.md
  agentfactory validate ./agents/code-reviewer.md`)
}
