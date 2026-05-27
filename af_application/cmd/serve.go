package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const serveBanner = `
┌─ AgentFactory ───────────────────┐
│       Platform Chat Server       │
└──────────────────────────────────┘
`

type tgUpdate struct {
	UpdateID int       `json:"update_id"`
	Message  tgMessage `json:"message"`
}

type tgMessage struct {
	MessageID int    `json:"message_id"`
	Text      string `json:"text"`
	Chat      tgChat `json:"chat"`
}

type tgChat struct {
	ID int64 `json:"id"`
}

type tgGetUpdatesResp struct {
	OK     bool       `json:"ok"`
	Result []tgUpdate `json:"result"`
}

// ── Spec interface types ──────────────────────────────────────────────────────

type specInterfaceAuth struct {
	Type   string `json:"type"`
	APIKey string `json:"api_key"`
}

type specInterfacePolling struct {
	Interval string `json:"interval"`
	Timeout  string `json:"timeout"`
}

type specInterfaceHTTP struct {
	Path string `json:"path"`
}

type specInterfaceExposure struct {
	HTTP *specInterfaceHTTP `json:"http"`
}

type specInterface struct {
	Type           string                 `json:"type"`
	Platform       string                 `json:"platform"`
	Mode           string                 `json:"mode"`
	PlatformConfig map[string]string      `json:"platform_config"`
	Polling        *specInterfacePolling  `json:"polling"`
	Exposure       *specInterfaceExposure `json:"exposure"`
	Authentication *specInterfaceAuth     `json:"authentication"`
}

type loadOutputServe struct {
	Success       bool            `json:"success"`
	Error         string          `json:"error"`
	AgentName     string          `json:"agent_name"`
	SpecVersion   string          `json:"spec_version"`
	ModelProvider string          `json:"model_provider"`
	ModelName     string          `json:"model_name"`
	Interfaces    []specInterface `json:"interfaces"`
}

// ── Entry point ───────────────────────────────────────────────────────────────

func serveAgent(pythonPath, appDir string, args []string) {
	if len(args) == 0 {
		fatalf("usage: agentfactory serve <agent.md> [--port 8080]")
	}

	specPath := args[0]
	port := "8080"
	for i := 1; i < len(args)-1; i++ {
		if args[i] == "--port" || args[i] == "-p" {
			port = args[i+1]
		}
	}

	if !filepath.IsAbs(specPath) {
		cwd, _ := os.Getwd()
		specPath = filepath.Join(cwd, specPath)
	}
	if _, err := os.Stat(specPath); err != nil {
		fatalf("spec file not found: %s", specPath)
	}

	// ── Validate spec before serving ─────────────────────────────────────────
	if !validateSpec(specPath) {
		fmt.Printf("\n  %sAborted.%s\n\n", colorDim, colorReset)
		os.Exit(1)
	}

	// Load spec with full interface list
	spinner("Loading agent spec")

	// Load .env so ${env:VAR} tokens resolve correctly in Go
	loadDotEnv(appDir)
	payload, _ := json.Marshal(runtimeInput{Phase: "load", SpecPath: specPath})
	loadOut, err := loadSpecFull(pythonPath, appDir, payload)
	if err != nil {
		fatalf("failed to load spec: %v", err)
	}
	if loadOut.Error != "" {
		fatalf("%s", loadOut.Error)
	}

	// Print header
	fmt.Printf("%s%s%s\n", colorCyan, serveBanner, colorReset)
	fmt.Printf("  %sAgent%s   %s%s%s\n", colorDim, colorReset, colorBold, loadOut.AgentName, colorReset)
	fmt.Printf("  %sModel%s   %s%s / %s%s\n\n", colorDim, colorReset, colorCyan, loadOut.ModelProvider, loadOut.ModelName, colorReset)

	// Find and start platformchat interfaces
	started := false
	for _, iface := range loadOut.Interfaces {
		if iface.Type != "platformchat" {
			continue
		}
		switch iface.Platform {
		case "telegram":
			botToken := resolveToken(iface.Authentication)
			if botToken == "" {
				fmt.Fprintf(os.Stderr, "%s⚠  Telegram bot token not set — check TELEGRAM_BOT_TOKEN in .env%s\n",
					colorYellow, colorReset)
				continue
			}
			switch iface.Mode {
			case "polling", "":
				interval := parseDuration(iface.Polling, 30*time.Second)
				go runTelegramPolling(pythonPath, appDir, specPath, botToken, interval)
				fmt.Printf("  %s✓%s  Telegram polling started (every %s)\n",
					colorGreen, colorReset, interval)
				started = true
			case "notification":
				path := "/telegram"
				if iface.Exposure != nil && iface.Exposure.HTTP != nil && iface.Exposure.HTTP.Path != "" {
					path = iface.Exposure.HTTP.Path
				}
				secretToken := ""
				if iface.PlatformConfig != nil {
					secretToken = iface.PlatformConfig["secret_token"]
				}
				http.HandleFunc(path, makeTelegramWebhookHandler(
					pythonPath, appDir, specPath, botToken, secretToken))
				fmt.Printf("  %s✓%s  Telegram webhook at %s\n", colorGreen, colorReset, path)
				started = true
			}
		}
	}

	if !started {
		fatalf("no active platformchat interfaces found in spec")
	}

	// Only start the HTTP server if at least one webhook handler was registered
	if needsHTTPServer(loadOut.Interfaces) {
		fmt.Printf("\n  %sListening on :%s%s\n\n", colorDim, port, colorReset)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			fatalf("server error: %v", err)
		}
	} else {
		// Polling mode — just block forever
		fmt.Printf("\n  %sPress Ctrl+C to stop.%s\n\n", colorDim, colorReset)
		select {}
	}
}

// ── Telegram polling ──────────────────────────────────────────────────────────

func runTelegramPolling(pythonPath, appDir, specPath, botToken string, interval time.Duration) {
	offset := 0
	apiBase := "https://api.telegram.org/bot" + botToken

	fmt.Printf("  %s⟳%s  Polling Telegram...\n", colorCyan, colorReset)

	for {
		url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=25", apiBase, offset)
		resp, err := http.Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s⚠  poll error: %v%s\n", colorYellow, err, colorReset)
			time.Sleep(interval)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var updates tgGetUpdatesResp
		if err := json.Unmarshal(body, &updates); err != nil || !updates.OK {
			time.Sleep(interval)
			continue
		}

		for _, upd := range updates.Result {
			offset = upd.UpdateID + 1
			text := strings.TrimSpace(upd.Message.Text)
			chatID := upd.Message.Chat.ID
			if text == "" {
				continue
			}

			fmt.Printf("  %s←%s  [%d] %s\n", colorDim, colorReset, chatID, text)
			response := invokeWebhookBridge(pythonPath, appDir, specPath, "telegram",
				strconv.FormatInt(chatID, 10), text)
			if response != "" {
				sendTelegramMessage(apiBase, chatID, response)
				fmt.Printf("  %s→%s  [%d] %s\n", colorGreen, colorReset, chatID, truncate(response, 80))
			}
		}

		time.Sleep(interval)
	}
}

// ── Telegram webhook handler ──────────────────────────────────────────────────

func makeTelegramWebhookHandler(pythonPath, appDir, specPath, botToken, secretToken string) http.HandlerFunc {
	apiBase := "https://api.telegram.org/bot" + botToken
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if secretToken != "" && r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != secretToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var upd tgUpdate
		if err := json.Unmarshal(body, &upd); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		text := strings.TrimSpace(upd.Message.Text)
		chatID := upd.Message.Chat.ID
		if text == "" {
			w.WriteHeader(http.StatusOK)
			return
		}
		fmt.Printf("  %s←%s  [%d] %s\n", colorDim, colorReset, chatID, text)
		response := invokeWebhookBridge(pythonPath, appDir, specPath, "telegram",
			strconv.FormatInt(chatID, 10), text)
		if response != "" {
			sendTelegramMessage(apiBase, chatID, response)
			fmt.Printf("  %s→%s  [%d] %s\n", colorGreen, colorReset, chatID, truncate(response, 80))
		}
		w.WriteHeader(http.StatusOK)
	}
}

// ── Telegram send ─────────────────────────────────────────────────────────────

func sendTelegramMessage(apiBase string, chatID int64, text string) {
	payload, _ := json.Marshal(map[string]any{"chat_id": chatID, "text": text})
	resp, err := http.Post(apiBase+"/sendMessage", "application/json", bytes.NewReader(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s⚠  sendMessage error: %v%s\n", colorYellow, err, colorReset)
		return
	}
	resp.Body.Close()
}

// ── Webhook bridge ────────────────────────────────────────────────────────────

func invokeWebhookBridge(pythonPath, appDir, specPath, platform, sessionID, userInput string) string {
	type webhookInput struct {
		Phase     string `json:"phase"`
		SpecPath  string `json:"spec_path"`
		Platform  string `json:"platform"`
		SessionID string `json:"session_id"`
		UserInput string `json:"user_input"`
	}
	payload, _ := json.Marshal(webhookInput{
		Phase:     "webhook",
		SpecPath:  specPath,
		Platform:  platform,
		SessionID: sessionID,
		UserInput: userInput,
	})
	script := filepath.Join(appDir, "adapters", "runtime_bridge.py")
	cmd := newPythonCmd(pythonPath, script, appDir, payload)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s⚠  bridge: %s%s\n",
			colorYellow, strings.TrimSpace(stderr.String()), colorReset)
		return ""
	}
	var out struct {
		Success           bool   `json:"success"`
		Error             string `json:"error"`
		AssistantResponse string `json:"assistant_response"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return ""
	}
	if !out.Success {
		fmt.Fprintf(os.Stderr, "%s⚠  %s%s\n", colorYellow, out.Error, colorReset)
		return ""
	}
	return out.AssistantResponse
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func loadSpecFull(pythonPath, appDir string, payload []byte) (loadOutputServe, error) {
	script := filepath.Join(appDir, "adapters", "runtime_bridge.py")
	cmd := newPythonCmd(pythonPath, script, appDir, payload)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return loadOutputServe{}, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
	}
	var out loadOutputServe
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return loadOutputServe{}, fmt.Errorf("parse: %w", err)
	}
	return out, nil
}

func newPythonCmd(pythonPath, script, appDir string, stdin []byte) *pythonCmd {
	return newExecCmd(pythonPath, script, appDir, stdin)
}

func resolveToken(auth *specInterfaceAuth) string {
	if auth == nil {
		return ""
	}
	val := auth.APIKey
	if strings.HasPrefix(val, "${env:") && strings.HasSuffix(val, "}") {
		return os.Getenv(val[6 : len(val)-1])
	}
	return val
}

func parseDuration(p *specInterfacePolling, def time.Duration) time.Duration {
	if p == nil {
		return def
	}
	s := strings.TrimSuffix(p.Interval, "s")
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return time.Duration(n) * time.Second
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// needsHTTPServer returns true if any platformchat interface uses notification
// mode, which requires an HTTP listener.
func needsHTTPServer(interfaces []specInterface) bool {
	for _, iface := range interfaces {
		if iface.Type == "platformchat" && iface.Mode == "notification" {
			return true
		}
	}
	return false
}

// loadDotEnv reads KEY=VALUE pairs from <appDir>/.env and sets them
// as environment variables if not already set.
func loadDotEnv(appDir string) {
	path := filepath.Join(appDir, ".env")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
