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
	"time"
)

const initBanner = `
 ┌──────────────────────────────────────┐
 │         agentfactory init            │
 │   Turn your idea into an agent spec  │
 └──────────────────────────────────────┘
`

// bridgeInput is the JSON payload sent to go_bridge.py via stdin.
// Phase "clarify" only needs Idea.
// Phase "generate" needs Idea + ClarificationQuestions + ClarificationAnswers.
type bridgeInput struct {
	Phase                  string   `json:"phase"`
	Idea                   string   `json:"idea"`
	ClarificationQuestions []string `json:"clarification_questions"`
	ClarificationAnswers   []string `json:"clarification_answers"`
	EnrichedIdea           string   `json:"enriched_idea"`
	AgentSpec              any      `json:"agent_spec"`
	ValidationErrors       []string `json:"validation_errors"`
	ExportedFile           string   `json:"exported_file"`
}

// bridgeOutput is the JSON response from go_bridge.py.
type bridgeOutput struct {
	Success                bool           `json:"success"`
	Error                  string         `json:"error"`
	ClarificationQuestions []string       `json:"clarification_questions"`
	AgentSpec              map[string]any `json:"agent_spec"`
	ValidationErrors       []string       `json:"validation_errors"`
	ExportedFile           string         `json:"exported_file"`
}

func runInit(pythonPath, appDir string) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("%s%s%s\n", colorCyan, initBanner, colorReset)
	fmt.Printf("  %sLet's build your agent spec step by step.%s\n\n",
		colorDim, colorReset)

	// ── Step 1: Collect idea from user ────────────────────────────────────────
	printStep(1, "Describe your agent idea")
	fmt.Printf("  %sExample: \"A support agent that handles refund requests\"%s\n\n",
		colorDim, colorReset)
	fmt.Printf("  %s→%s ", colorCyan, colorReset)

	idea := readLine(reader)
	if idea == "" {
		fatalf("idea cannot be empty")
	}

	// ── Step 2: Ask Python to generate clarification questions ────────────────
	fmt.Println()
	spinner("Thinking about your idea")

	clarifyResult, err := invokeBridge(pythonPath, appDir, bridgeInput{
		Phase:                  "clarify",
		Idea:                   idea,
		ClarificationQuestions: []string{},
		ClarificationAnswers:   []string{},
		ValidationErrors:       []string{},
	})
	if err != nil {
		fatalf("bridge error: %v", err)
	}
	if clarifyResult.Error != "" {
		fatalf("%s", clarifyResult.Error)
	}

	questions := clarifyResult.ClarificationQuestions

	// ── Step 3: Present questions, collect answers from user ──────────────────
	fmt.Printf("\n  %s%s✦  A few quick questions to sharpen the spec:%s\n",
		colorBold, colorCyan, colorReset)
	fmt.Printf("  %s(press Enter to skip any question)%s\n\n",
		colorDim, colorReset)

	answers := make([]string, len(questions))
	for i, q := range questions {
		fmt.Printf("  %s%d.%s %s\n", colorBold, i+1, colorReset, q)
		fmt.Printf("     %s→%s ", colorCyan, colorReset)
		answers[i] = readLine(reader)
		fmt.Println()
	}

	// ── Step 4: Send idea + answers to Python to generate the spec ────────────
	spinner("Generating your agent spec")

	result, err := invokeBridge(pythonPath, appDir, bridgeInput{
		Phase:                  "generate",
		Idea:                   idea,
		ClarificationQuestions: questions,
		ClarificationAnswers:   answers,
		ValidationErrors:       []string{},
	})
	if err != nil {
		fatalf("bridge error: %v", err)
	}
	if result.Error != "" {
		fatalf("%s", result.Error)
	}

	// ── Step 5: Render result ─────────────────────────────────────────────────
	fmt.Println()
	fmt.Println("  " + strings.Repeat("─", 50))

	if len(result.ValidationErrors) > 0 {
		fmt.Printf("\n  %s%s⚠  Validation warnings%s\n", colorBold, colorYellow, colorReset)
		for _, e := range result.ValidationErrors {
			fmt.Printf("     %s•%s %s\n", colorYellow, colorReset, e)
		}
	}

	if result.ExportedFile == "" {
		fmt.Printf("\n  %s%s✗  Export failed — no file generated%s\n\n",
			colorBold, colorRed, colorReset)
		os.Exit(1)
	}

	absPath, _ := filepath.Abs(filepath.Join(appDir, result.ExportedFile))

	fmt.Printf("\n  %s%s✓  Agent spec created%s\n\n", colorBold, colorGreen, colorReset)
	fmt.Printf("  %sFile:%s  %s\n", colorDim, colorReset, absPath)

	if spec := result.AgentSpec; spec != nil {
		printSpecSummary(spec)
	}

	fmt.Printf("\n  %sNext step:%s\n", colorDim, colorReset)
	fmt.Printf("  %s→%s agentfactory vali %s\n\n",
		colorCyan, colorReset, result.ExportedFile)
}

// printSpecSummary renders a compact table of key spec fields.
func printSpecSummary(spec map[string]any) {
	fmt.Printf("\n  %sSpec Summary%s\n", colorBold, colorReset)
	fmt.Println("  " + strings.Repeat("─", 40))

	rows := []struct{ label, key string }{
		{"Name", "name"},
		{"Description", "description"},
		{"Version", "version"},
		{"Execution", "execution_mode"},
		{"Output", "output_format"},
		{"Memory", "memory_type"},
		{"Iterations", "max_iterations"},
	}
	for _, r := range rows {
		val, ok := spec[r.key]
		if !ok || val == nil || val == "" {
			continue
		}
		fmt.Printf("  %s%-14s%s %v\n", colorDim, r.label, colorReset, val)
	}

	if tools, ok := spec["suggested_tools"].([]any); ok && len(tools) > 0 {
		names := make([]string, 0, len(tools))
		for _, t := range tools {
			if s, ok := t.(string); ok {
				names = append(names, s)
			}
		}
		fmt.Printf("  %s%-14s%s %s\n", colorDim, "Tools", colorReset,
			strings.Join(names, ", "))
	}
}

// invokeBridge spawns go_bridge.py, sends JSON via stdin, returns parsed output.
func invokeBridge(pythonPath, appDir string, input bridgeInput) (*bridgeOutput, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	script := filepath.Join(appDir, "adapters", "go_bridge.py")
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

	var out bridgeOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fmt.Errorf("parse output: %w\nraw: %s", err, stdout.String())
	}
	return &out, nil
}

// spinner shows an animated spinner then clears it with a ✓.
func spinner(label string) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	for i := 0; i < 14; i++ {
		fmt.Printf("\r  %s%s%s  %s...",
			colorCyan, frames[i%len(frames)], colorReset, label)
		time.Sleep(80 * time.Millisecond)
	}
	fmt.Printf("\r  %s✓%s  %s          \n", colorGreen, colorReset, label)
}

// printStep prints a numbered step header.
func printStep(n int, label string) {
	fmt.Printf("  %s%sStep %d%s  %s\n",
		colorBold, colorCyan, n, colorReset, label)
}

// readLine reads a trimmed line from stdin.
func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
