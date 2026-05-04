package tools

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

// DockerTool provides docker management on brain server.
func DockerTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Docker",
			Description: "Manage Docker containers on brain server via SSH. Commands: ps, logs, restart, stats, compose.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Docker subcommand (e.g. 'ps', 'logs aurora --tail 50', 'restart aurora', 'compose up -d')",
					},
				},
				"required": []string{"command"},
			},
		},
		Execute: func(input map[string]interface{}) string {
			command, _ := input["command"].(string)
			if command == "" {
				return "Error: command required"
			}

			// Prepend 'docker' if not already there
			if !strings.HasPrefix(command, "docker") {
				if strings.HasPrefix(command, "compose") {
					command = "docker " + command
				} else {
					command = "docker " + command
				}
			}

			cmd := exec.Command("ssh", SSHHost, command)
			output, err := cmd.CombinedOutput()
			result := string(output)
			if err != nil {
				result += fmt.Sprintf("\nError: %v", err)
			}
			if len(result) > 30000 {
				result = result[:30000] + "\n... (truncated)"
			}
			if result == "" {
				return "(empty output)"
			}
			return strings.TrimRight(result, "\n")
		},
	}
}

// ClipboardTool reads/writes system clipboard.
func ClipboardTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Clipboard",
			Description: "Read or write system clipboard. Use action='read' to get content, action='write' with text to set it.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type": "string",
						"enum": []string{"read", "write"},
					},
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Text to write to clipboard (only for write action)",
					},
				},
				"required": []string{"action"},
			},
		},
		Execute: func(input map[string]interface{}) string {
			action, _ := input["action"].(string)

			if runtime.GOOS != "windows" {
				return "Clipboard only supported on Windows"
			}

			if action == "read" {
				cmd := exec.Command("powershell", "-Command", "Get-Clipboard")
				output, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Sprintf("Error: %v", err)
				}
				result := strings.TrimRight(string(output), "\r\n")
				if result == "" {
					return "(clipboard empty)"
				}
				return result
			}

			if action == "write" {
				text, _ := input["text"].(string)
				if text == "" {
					return "Error: text required for write"
				}
				cmd := exec.Command("powershell", "-Command",
					fmt.Sprintf("Set-Clipboard -Value '%s'", strings.ReplaceAll(text, "'", "''")))
				if err := cmd.Run(); err != nil {
					return fmt.Sprintf("Error: %v", err)
				}
				return fmt.Sprintf("Written %d chars to clipboard", len(text))
			}

			return "Error: action must be 'read' or 'write'"
		},
	}
}

// ProcessTool manages system processes.
func ProcessTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Process",
			Description: "Manage system processes. Actions: list (ps), kill PID, find NAME.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type": "string",
						"enum": []string{"list", "kill", "find"},
					},
					"target": map[string]interface{}{
						"type":        "string",
						"description": "PID to kill, or process name to find",
					},
				},
				"required": []string{"action"},
			},
		},
		Execute: func(input map[string]interface{}) string {
			action, _ := input["action"].(string)
			target, _ := input["target"].(string)

			switch action {
			case "list":
				var cmd *exec.Cmd
				if runtime.GOOS == "windows" {
					cmd = exec.Command("powershell", "-Command", "Get-Process | Sort-Object CPU -Descending | Select-Object -First 20 Id, ProcessName, CPU, WorkingSet | Format-Table -AutoSize")
				} else {
					cmd = exec.Command("ps", "aux", "--sort=-pcpu")
				}
				output, _ := cmd.CombinedOutput()
				return string(output)

			case "kill":
				if target == "" {
					return "Error: target PID required"
				}
				var cmd *exec.Cmd
				if runtime.GOOS == "windows" {
					cmd = exec.Command("taskkill", "/PID", target, "/F")
				} else {
					cmd = exec.Command("kill", target)
				}
				output, err := cmd.CombinedOutput()
				if err != nil {
					return fmt.Sprintf("Error: %v\n%s", err, output)
				}
				return string(output)

			case "find":
				if target == "" {
					return "Error: target name required"
				}
				var cmd *exec.Cmd
				if runtime.GOOS == "windows" {
					cmd = exec.Command("powershell", "-Command",
						fmt.Sprintf("Get-Process | Where-Object { $_.ProcessName -like '*%s*' } | Format-Table Id, ProcessName, CPU -AutoSize", target))
				} else {
					cmd = exec.Command("pgrep", "-a", target)
				}
				output, _ := cmd.CombinedOutput()
				result := strings.TrimSpace(string(output))
				if result == "" {
					return "No processes found matching: " + target
				}
				return result

			default:
				return "Error: action must be list, kill, or find"
			}
		},
	}
}

// ServerDashboard shows brain server stats.
func ServerDashboardTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "ServerStatus",
			Description: "Get brain server status: CPU, RAM, GPU, disk, containers, model. Quick health check.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"section": map[string]interface{}{
						"type":        "string",
						"description": "Section: all, cpu, gpu, disk, docker, model (default: all)",
					},
				},
			},
		},
		Execute: func(input map[string]interface{}) string {
			section, _ := input["section"].(string)
			if section == "" {
				section = "all"
			}

			var commands []string
			switch section {
			case "cpu":
				commands = []string{"uptime && free -h"}
			case "gpu":
				commands = []string{"nvidia-smi --query-gpu=name,memory.used,memory.total,temperature.gpu,utilization.gpu --format=csv,noheader"}
			case "disk":
				commands = []string{"df -h /data"}
			case "docker":
				commands = []string{"docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"}
			case "model":
				commands = []string{"curl -s http://localhost:8100/v1/models 2>/dev/null | python3 -m json.tool 2>/dev/null || echo 'Model server not responding'"}
			default:
				commands = []string{
					"echo '=== SYSTEM ===' && uptime && free -h | head -2",
					"echo '=== GPU ===' && nvidia-smi --query-gpu=name,memory.used,memory.total,temperature.gpu --format=csv,noheader 2>/dev/null || echo 'nvidia-smi not available'",
					"echo '=== DISK ===' && df -h /data | tail -1",
					"echo '=== DOCKER ===' && docker ps --format '{{.Names}}: {{.Status}}' 2>/dev/null | head -10",
				}
			}

			fullCmd := strings.Join(commands, " && ")
			cmd := exec.Command("ssh", SSHHost, fullCmd)
			output, err := cmd.CombinedOutput()
			result := string(output)
			if err != nil {
				result += fmt.Sprintf("\nError: %v", err)
			}
			return strings.TrimRight(result, "\n")
		},
	}
}

// ModelSwitchTool switches LLM model on brain.
func ModelSwitchTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "ModelSwitch",
			Description: "Switch LLM model on brain server. Options: q4 (fast, current), q6 (better quality), status (show current).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"model": map[string]interface{}{
						"type":        "string",
						"description": "Model preset: q4, q6, status",
					},
				},
				"required": []string{"model"},
			},
		},
		Execute: func(input map[string]interface{}) string {
			model, _ := input["model"].(string)
			if model == "" {
				return "Error: model required (q4, q6, status)"
			}

			cmd := exec.Command("ssh", "brain", "/data/llama.cpp/model-switch.sh "+model)
			output, err := cmd.CombinedOutput()
			result := string(output)
			if err != nil {
				result += fmt.Sprintf("\nError: %v", err)
			}
			return strings.TrimRight(result, "\n")
		},
	}
}
