package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/madgodinc/aurora-cli/internal/agent"
	"github.com/madgodinc/aurora-cli/internal/config"
	"github.com/madgodinc/aurora-cli/internal/tools"
	"github.com/madgodinc/aurora-cli/internal/ui"

	tea "charm.land/bubbletea/v2"
)

const version = "0.2.0"

func main() {
	defer tools.CleanupChildren()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		tools.CleanupChildren()
		os.Exit(0)
	}()

	if len(os.Args) > 1 && os.Args[1] == "--version" {
		cfg := config.Load()
		fmt.Printf("Aurora CLI v%s — Mad God Inc.\n", version)
		fmt.Printf("Tools: %d | Memory Palace | Streaming\n", tools.ToolCount(cfg.HasSSH))
		if cfg.HasSSH {
			fmt.Println("Server: ✓ SSH connected")
		}
		os.Exit(0)
	}

	// Load or setup config
	cfg := config.Load()
	if cfg.NeedsSetup() {
		cfg = config.RunSetup()
	}

	// Check for updates (non-blocking, max every 6h)
	config.CheckForUpdate(version)

	// Non-interactive mode
	if len(os.Args) > 2 && (os.Args[1] == "-p" || os.Args[1] == "--print") {
		prompt := strings.Join(os.Args[2:], " ")
		runPrint(cfg, prompt)
		return
	}

	// Interactive TUI
	p := tea.NewProgram(ui.NewModel(version, cfg))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runPrint(cfg *config.Config, prompt string) {
	agCfg := agent.Config{
		ProxyURL: cfg.ProxyURL,
		APIKey:   cfg.APIKey,
		Model:    cfg.Model,
		WorkDir:  mustGetwd(),
		Username: cfg.Username,
		HasSSH:   cfg.HasSSH,
		Token:    cfg.Token,
		SSHAlias: cfg.SSHAlias,
	}
	ag := agent.New(agCfg)
	ch := ag.Events()

	go ag.Run(prompt)

	for ev := range ch {
		switch ev.Type {
		case "text":
			fmt.Print(ev.Text)
		case "tool_start":
			fmt.Printf("\n⚡ %s %s\n", ev.ToolName, ev.ToolInput)
		case "tool_done":
			fmt.Printf("✓ %s %s\n", ev.ToolName, ev.ToolResult)
		case "error":
			fmt.Fprintf(os.Stderr, "Error: %s\n", ev.Text)
		case "done":
			fmt.Println()
			return
		}
	}
}

func mustGetwd() string {
	wd, _ := os.Getwd()
	return wd
}
