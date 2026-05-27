package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// validateSpec runs the af_cli validator against the spec file before
// the runtime starts. Returns true if the user wants to proceed.
//
// Search order for the validator binary:
//  1. AF_VALIDATOR env var
//  2. ../af_cli/dist/agentfactory.exe  (sibling directory layout)
//  3. agentfactory-vali on PATH
func validateSpec(specPath string) bool {
	validatorPath := resolveValidator()
	if validatorPath == "" {
		// Validator not found — skip silently, don't block the user
		return true
	}

	fmt.Printf("  %s⟳%s  Validating spec...\n", colorDim, colorReset)

	cmd := exec.Command(validatorPath, "vali", specPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()

	output := out.String()
	hasErrors := err != nil // af_cli exits 1 on errors

	if !hasErrors {
		fmt.Printf("  %s✓%s  Spec is valid\n\n", colorGreen, colorReset)
		return true
	}

	// Print the validator output
	fmt.Println()
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fmt.Printf("  %s\n", line)
	}
	fmt.Println()

	// Ask user whether to proceed despite errors
	fmt.Printf("  %s⚠  Spec has validation errors. Continue anyway? [y/N]%s ", colorYellow, colorReset)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	return answer == "y" || answer == "yes"
}

// resolveValidator finds the af_cli validator binary.
func resolveValidator() string {
	// 1. Explicit env override
	if v := os.Getenv("AF_VALIDATOR"); v != "" {
		if _, err := os.Stat(v); err == nil {
			return v
		}
	}

	// 2. Sibling af_cli/dist/ directory relative to the running binary
	exe, err := os.Executable()
	if err != nil {
		exe, _ = filepath.Abs(os.Args[0])
	}

	binaryName := "agentfactory"
	if runtime.GOOS == "windows" {
		binaryName = "agentfactory.exe"
	}

	candidates := []string{
		// binary is in af_application/ → sibling is af_cli/
		filepath.Join(filepath.Dir(exe), "..", "af_cli", "dist", binaryName),
		// binary is in af_application/cmd/ → go up two levels
		filepath.Join(filepath.Dir(exe), "..", "..", "af_cli", "dist", binaryName),
		// cwd-relative (useful during development)
		filepath.Join("..", "af_cli", "dist", binaryName),
	}

	for _, c := range candidates {
		abs, _ := filepath.Abs(c)
		if _, err := os.Stat(abs); err == nil {
			return abs
		}
	}

	// 3. PATH fallback
	if p, err := exec.LookPath("agentfactory-vali"); err == nil {
		return p
	}

	return "" // not found — skip validation
}
