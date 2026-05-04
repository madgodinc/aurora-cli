package tools

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

// SSHHost is the SSH alias for remote server. Set by agent on init.
var SSHHost = "brain"

// Track child processes for cleanup
var (
	childProcs   []*exec.Cmd
	childProcsMu sync.Mutex
)

// CleanupChildren kills all tracked child processes.
func CleanupChildren() {
	childProcsMu.Lock()
	defer childProcsMu.Unlock()
	for _, cmd := range childProcs {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
	}
	childProcs = nil
}

func trackChild(cmd *exec.Cmd) {
	childProcsMu.Lock()
	childProcs = append(childProcs, cmd)
	childProcsMu.Unlock()
}

func untrackChild(cmd *exec.Cmd) {
	childProcsMu.Lock()
	for i, c := range childProcs {
		if c == cmd {
			childProcs = append(childProcs[:i], childProcs[i+1:]...)
			break
		}
	}
	childProcsMu.Unlock()
}

// BashTool creates the Bash tool.
func BashTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Bash",
			Description: "Execute a shell command. Returns stdout+stderr. Use for git, npm, pip, builds, tests.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "The bash command to execute",
					},
					"timeout": map[string]interface{}{
						"type":        "integer",
						"description": "Timeout in seconds (default 120)",
					},
				},
				"required": []string{"command"},
			},
		},
		Execute: executeBash,
	}
}

func executeBash(input map[string]interface{}) string {
	command, _ := input["command"].(string)
	if command == "" {
		return "Error: empty command"
	}

	timeoutSec := 120
	if t, ok := input["timeout"].(float64); ok && t > 0 {
		timeoutSec = int(t)
		if timeoutSec > 600 {
			timeoutSec = 600
		}
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		gitBash := `C:\Program Files\Git\bin\bash.exe`
		cmd = exec.Command(gitBash, "-c", command)
	} else {
		cmd = exec.Command("bash", "-c", command)
	}

	trackChild(cmd)
	defer untrackChild(cmd)

	done := make(chan error, 1)
	var output []byte
	var cmdErr error

	go func() {
		output, cmdErr = cmd.CombinedOutput()
		done <- cmdErr
	}()

	select {
	case <-done:
		result := string(output)
		if cmdErr != nil {
			result += fmt.Sprintf("\nExit code: %v", cmdErr)
		}
		if len(result) > 50000 {
			result = result[:50000] + "\n... (truncated)"
		}
		if result == "" {
			result = "(empty output)"
		}
		return result
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return fmt.Sprintf("Command timed out after %ds", timeoutSec)
	}
}

// RemoteShellTool creates the SSH remote shell tool.
func RemoteShellTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "RemoteShell",
			Description: "Execute command on brain server (192.168.0.100) via SSH.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Command for brain server",
					},
				},
				"required": []string{"command"},
			},
		},
		Execute: func(input map[string]interface{}) string {
			command, _ := input["command"].(string)
			if command == "" {
				return "Error: empty command"
			}
			cmd := exec.Command("ssh", SSHHost, command)
			output, err := cmd.CombinedOutput()
			result := string(output)
			if err != nil {
				result += fmt.Sprintf("\nError: %v", err)
			}
			if len(result) > 50000 {
				result = result[:50000] + "\n... (truncated)"
			}
			if result == "" {
				result = "(empty output)"
			}
			return strings.TrimRight(result, "\n")
		},
	}
}
