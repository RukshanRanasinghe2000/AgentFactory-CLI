package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runtimeInput is the JSON payload sent to the runtime bridge.
type runtimeInput struct {
	Phase     string              `json:"phase"`
	SpecPath  string              `json:"spec_path"`
	UserInput string              `json:"user_input"`
	Messages  []map[string]string `json:"messages"`
}

// runtimeOutput is the JSON response from the runtime bridge.
type runtimeOutput struct {
	Success           bool                `json:"success"`
	Error             string              `json:"error"`
	AgentName         string              `json:"agent_name"`
	SpecVersion       string              `json:"spec_version"`
	ModelProvider     string              `json:"model_provider"`
	ModelName         string              `json:"model_name"`
	AssistantResponse string              `json:"assistant_response"`
	ToolCalls         []map[string]any    `json:"tool_calls"`
	Messages          []map[string]string `json:"messages"`
}

func runAgent(pythonPath, appDir string, args []string) {
	if len(args) == 0 {
		fatalf("usage: agentfactory run <agent.md>")
	}

	specPath := args[0]

	// Resolve spec path relative to cwd if not absolute
	if !filepath.IsAbs(specPath) {
		cwd, _ := os.Getwd()
		specPath = filepath.Join(cwd, specPath)
	}

	if _, err := os.Stat(specPath); err != nil {
		fatalf("spec file not found: %s", specPath)
	}

	// ── Load spec info from Python ────────────────────────────────────────────
	spinner("Loading agent spec")

	loadResult, err := invokeRuntimeBridge(pythonPath, appDir, runtimeInput{
		Phase:    "load",
		SpecPath: specPath,
	})
	if err != nil {
		fatalf("failed to load spec: %v", err)
	}
	if loadResult.Error != "" {
		fatalf("%s", loadResult.Error)
	}

	// ── Print agent header ────────────────────────────────────────────────────
	fmt.Println()
	fmt.Printf(" %s┌─ AgentFactory ───────────────────┐%s\n", colorCyan, colorReset)
	fmt.Printf(" %s│%s %-34s%s│%s\n", colorCyan, colorBold, "Runtime", colorReset+colorCyan, colorReset)
	fmt.Printf(" %s└──────────────────────────────────┘%s\n", colorCyan, colorReset)
	fmt.Println()

	specVer := loadResult.SpecVersion
	if specVer == "" {
		specVer = "—"
	}
	fmt.Printf("  %sSpec Version%s   %s\n", colorDim, colorReset, specVer)
	fmt.Printf("  %sAgent%s          %s%s%s\n", colorDim, colorReset, colorBold, loadResult.AgentName, colorReset)
	fmt.Printf("  %sModel%s          %s%s / %s%s\n\n", colorDim, colorReset, colorCyan, loadResult.ModelProvider, loadResult.ModelName, colorReset)

	fmt.Printf("  %sType your message. Press Ctrl+C to exit.%s\n\n",
		colorDim, colorReset)

	// ── Chat loop ─────────────────────────────────────────────────────────────
	reader := bufio.NewReader(os.Stdin)
	var messages []map[string]string

	for {
		fmt.Printf("  %sYou%s  → ", colorBold, colorReset)
		userInput := readLine(reader)

		if userInput == "" {
			continue
		}
		if userInput == "/exit" || userInput == "/quit" {
			fmt.Printf("\n  %sBye!%s\n\n", colorDim, colorReset)
			break
		}

		spinner("Thinking")

		chatResult, err := invokeRuntimeBridge(pythonPath, appDir, runtimeInput{
			Phase:     "chat",
			SpecPath:  specPath,
			UserInput: userInput,
			Messages:  messages,
		})
		if err != nil {
			fmt.Printf("\n  %s✗  Error: %v%s\n\n", colorRed, err, colorReset)
			continue
		}
		if chatResult.Error != "" {
			fmt.Printf("\n  %s✗  %s%s\n\n", colorRed, chatResult.Error, colorReset)
			continue
		}

		// Print tool calls if any
		if len(chatResult.ToolCalls) > 0 {
			fmt.Printf("  %s⚙  Tools called:%s", colorDim, colorReset)
			for _, tc := range chatResult.ToolCalls {
				fmt.Printf(" %s%v%s", colorCyan, tc["tool"], colorReset)
			}
			fmt.Println()
		}

		// Print assistant response
		fmt.Println()
		fmt.Printf("  %sAgent%s → ", colorBold+colorGreen, colorReset)
		fmt.Println()
		for _, line := range strings.Split(chatResult.AssistantResponse, "\n") {
			fmt.Printf("    %s\n", line)
		}
		fmt.Println()

		// Carry conversation forward
		messages = chatResult.Messages
	}
}

// invokeRuntimeBridge calls the runtime bridge Python script.
func invokeRuntimeBridge(pythonPath, appDir string, input runtimeInput) (*runtimeOutput, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	script := filepath.Join(appDir, "adapters", "runtime_bridge.py")
	cmd := exec.Command(pythonPath, script)
	cmd.Dir = appDir
	cmd.Stdin = bytes.NewReader(payload)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("python exited: %w", err)
	}

	var out runtimeOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("parse output: %w\nraw: %s", err, stdout.String())
	}
	return &out, nil
}
