package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Config holds all Aurora CLI configuration.
type Config struct {
	ProxyURL   string `json:"proxy_url"`
	BackendURL string `json:"backend_url"`
	APIKey     string `json:"api_key"`
	Model      string `json:"model"`
	Username   string `json:"username"`
	Token      string `json:"token"`
	UserID     int    `json:"user_id"`
	SSHAlias   string `json:"ssh_alias"`   // SSH host alias (default: "brain")

	// Auto-detected
	HasSSH  bool `json:"-"`
	IsOwner bool `json:"-"`
}

var configDir string
var configFile string

func init() {
	home, _ := os.UserHomeDir()
	configDir = filepath.Join(home, ".aurora")
	configFile = filepath.Join(configDir, "cli_config.json")
	os.MkdirAll(configDir, 0755)
}

// Load reads config from file, env, or defaults.
func Load() *Config {
	cfg := &Config{
		ProxyURL:   "https://llm.fraylon.net",
		BackendURL: "https://llm.fraylon.net",
		APIKey:     "aurora-brain-2026",
		Model:      "claude-sonnet-4-20250514",
		Username:   "User",
		SSHAlias:   "brain",
	}

	// Load from file
	if data, err := os.ReadFile(configFile); err == nil {
		json.Unmarshal(data, cfg)
	}

	// Env overrides
	if v := os.Getenv("ANTHROPIC_BASE_URL"); v != "" {
		cfg.ProxyURL = v
	}
	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("AURORA_USERNAME"); v != "" {
		cfg.Username = v
	}

	// Auto-detect SSH
	cfg.HasSSH = checkSSH()
	cfg.IsOwner = cfg.HasSSH

	// If local network, set backend URL to direct brain
	if strings.Contains(cfg.ProxyURL, "192.168.0.100") {
		cfg.BackendURL = "http://192.168.0.100:8080"
	}

	return cfg
}

// Save writes config to file.
func (c *Config) Save() error {
	os.MkdirAll(configDir, 0755)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

// NeedsSetup returns true if no token.
func (c *Config) NeedsSetup() bool {
	return c.Token == "" && !c.IsOwner
}

// IsAuthenticated returns true if user has a valid token or is owner.
func (c *Config) IsAuthenticated() bool {
	return c.IsOwner || c.Token != ""
}

// RunSetup runs interactive first-time setup.
func RunSetup() *Config {
	cfg := &Config{
		Model: "claude-sonnet-4-20250514",
	}

	fmt.Println()
	fmt.Println("  Aurora CLI -- First Time Setup")
	fmt.Println("  --------------------------------")
	fmt.Println()
	fmt.Println("  Connect to:")
	fmt.Println("  1) Aurora Brain -- local (192.168.0.100)")
	fmt.Println("  2) Aurora Brain -- internet (llm.fraylon.net)")
	fmt.Println("  3) Your own server (any Anthropic-compatible API)")
	fmt.Print("  Choice [1/2/3]: ")

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		cfg.ProxyURL = "http://192.168.0.100:8082"
		cfg.BackendURL = "http://192.168.0.100:8080"
	case "3":
		fmt.Print("  Server URL (e.g. http://192.168.1.50:8082): ")
		fmt.Scanln(&cfg.ProxyURL)
		cfg.BackendURL = cfg.ProxyURL
		fmt.Print("  API key (or press Enter for default): ")
		var customKey string
		fmt.Scanln(&customKey)
		if customKey != "" {
			cfg.APIKey = customKey
		}
		fmt.Print("  SSH alias for server tools (or press Enter to skip): ")
		var sshAlias string
		fmt.Scanln(&sshAlias)
		if sshAlias != "" {
			// TODO: store custom SSH alias
			fmt.Println("  \033[33mCustom SSH not yet implemented, using 'brain'\033[0m")
		}
	default:
		cfg.ProxyURL = "https://llm.fraylon.net"
		cfg.BackendURL = "https://llm.fraylon.net"
	}

	cfg.APIKey = "aurora-brain-2026"

	// Check SSH
	cfg.HasSSH = checkSSH()
	cfg.IsOwner = cfg.HasSSH

	if cfg.IsOwner {
		// Owner doesn't need TG auth
		fmt.Println()
		fmt.Println("  [OK] SSH to brain detected -- owner mode")
		cfg.Username = "Mad God"
		cfg.Token = "owner"
		cfg.Save()
		fmt.Println("  [OK] Config saved!")
		fmt.Println()
		return cfg
	}

	// TG auth for non-owners
	fmt.Println()
	fmt.Println("  Authenticating via Telegram...")

	token, username, userID := doTelegramAuth(cfg.BackendURL)
	if token != "" {
		cfg.Token = token
		cfg.Username = username
		cfg.UserID = userID
		cfg.Save()
		fmt.Printf("\n  [OK] Welcome, %s! Config saved.\n\n", username)
	} else {
		fmt.Println("\n  [FAIL] Auth failed. You can retry with /login")
		cfg.Username = "Guest"
		cfg.Save()
	}

	return cfg
}

func doTelegramAuth(backendURL string) (token, username string, userID int) {
	// 1. Get auth code
	resp, err := http.Post(backendURL+"/api/auth/cli/start", "application/json", nil)
	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return "", "", 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var startData struct {
		Code         string `json:"code"`
		BotUsername  string `json:"bot_username"`
		Instructions string `json:"instructions"`
	}
	json.Unmarshal(body, &startData)

	if startData.Code == "" {
		fmt.Println("  Failed to get auth code")
		return "", "", 0
	}

	// 2. Show instructions
	fmt.Println()
	fmt.Printf("  Send this code to @%s in Telegram:\n\n", startData.BotUsername)
	fmt.Printf("       %s\n\n", startData.Code)
	fmt.Print("  Waiting for confirmation...")

	// 3. Poll
	client := &http.Client{Timeout: 10 * time.Second}
	for i := 0; i < 60; i++ { // 5 min max
		time.Sleep(5 * time.Second)
		fmt.Print(".")

		pollResp, err := client.Get(fmt.Sprintf("%s/api/auth/cli/poll?code=%s", backendURL, startData.Code))
		if err != nil {
			continue
		}
		pollBody, _ := io.ReadAll(pollResp.Body)
		pollResp.Body.Close()

		var pollData struct {
			Status    string `json:"status"`
			Token     string `json:"token"`
			UserID    int    `json:"user_id"`
			Username  string `json:"username"`
			FirstName string `json:"first_name"`
		}
		json.Unmarshal(pollBody, &pollData)

		if pollData.Status == "ok" {
			name := pollData.FirstName
			if name == "" {
				name = pollData.Username
			}
			fmt.Printf("\n  [OK] Authenticated!")
			return pollData.Token, name, pollData.UserID
		}
	}

	fmt.Println("\n  Timeout — code expired.")
	return "", "", 0
}

func checkSSH() bool {
	cmd := exec.Command("ssh", "-o", "ConnectTimeout=3", "-o", "BatchMode=yes", "brain", "echo ok")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return len(output) > 0 && output[0] == 'o'
}
