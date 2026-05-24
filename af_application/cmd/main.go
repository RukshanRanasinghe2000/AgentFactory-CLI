package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

const appBanner = `
 ┌──────────────────────────────────────┐
 │        AgentFactory CLI              │
 │   Build agents from ideas            │
 └──────────────────────────────────────┘
`

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	pythonPath, appDir := resolvePython()

	switch os.Args[1] {
	case "init":
		runInit(pythonPath, appDir)

	case "version", "--version", "-v":
		fmt.Printf("%sagentfactory-app v0.1.0%s\n", colorCyan, colorReset)

	case "help", "--help", "-h":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "%serror:%s unknown command %q\n\n",
			colorRed, colorReset, os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// resolvePython locates the Python interpreter and the af_application root.
// Priority: .venv inside af_application → system python3 → system python.
func resolvePython() (pythonPath, appDir string) {
	// af_application is the parent of cmd/
	exe, err := os.Executable()
	if err != nil {
		exe, _ = filepath.Abs(os.Args[0])
	}

	// Walk up from the binary location to find af_application
	dir := filepath.Dir(exe)
	for i := 0; i < 4; i++ {
		if isAppDir(dir) {
			appDir, _ = filepath.Abs(dir)
			break
		}
		dir = filepath.Dir(dir)
	}

	// Fallback: assume cwd is af_application
	if appDir == "" {
		cwd, _ := os.Getwd()
		if isAppDir(cwd) {
			appDir = cwd
		} else {
			parent := filepath.Dir(cwd)
			if isAppDir(parent) {
				appDir = parent
			}
		}
	}

	if appDir == "" {
		fmt.Fprintln(os.Stderr, colorRed+"error:"+colorReset+" cannot locate af_application directory")
		os.Exit(1)
	}

	// Prefer venv python
	venv := filepath.Join(appDir, ".venv", "Scripts", "python.exe")
	if runtime.GOOS != "windows" {
		venv = filepath.Join(appDir, ".venv", "bin", "python")
	}
	if _, err := os.Stat(venv); err == nil {
		return venv, appDir
	}

	// Fall back to system python
	for _, name := range []string{"python3", "python"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, appDir
		}
	}

	fmt.Fprintln(os.Stderr, colorRed+"error:"+colorReset+" python not found — run: uv venv && uv sync")
	os.Exit(1)
	return
}

// isAppDir returns true if the directory looks like af_application root.
func isAppDir(dir string) bool {
	markers := []string{"graphs", "nodes", "adapters", "prompts"}
	for _, m := range markers {
		if _, err := os.Stat(filepath.Join(dir, m)); err != nil {
			return false
		}
	}
	return true
}

func printUsage() {
	fmt.Printf("%s%s%s", colorCyan, appBanner, colorReset)
	fmt.Println(`Usage:
  agentfactory init          Create a new agent spec interactively
  agentfactory version       Show version
  agentfactory help          Show this help

Examples:
  agentfactory init`)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, colorRed+"error: "+colorReset+format+"\n", args...)
	os.Exit(1)
}
