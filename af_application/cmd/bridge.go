package main

import (
	"bytes"
	"os/exec"
)

// pythonCmd is an alias so serve.go can reference it without importing os/exec directly.
type pythonCmd = exec.Cmd

// newExecCmd creates a Python subprocess with the given script, working dir, and stdin payload.
func newExecCmd(pythonPath, script, appDir string, stdin []byte) *exec.Cmd {
	cmd := exec.Command(pythonPath, script)
	cmd.Dir = appDir
	cmd.Stdin = bytes.NewReader(stdin)
	return cmd
}
