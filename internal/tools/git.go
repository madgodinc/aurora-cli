package tools

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

// GitTool provides native git operations.
func GitTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name: "Git",
			Description: "Execute git commands. Supports: status, diff, log, add, commit, branch, checkout, push, pull, stash.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Git subcommand and args (e.g. 'status', 'diff --cached', 'log --oneline -10')",
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

			// Block dangerous commands without explicit intent
			dangerous := []string{"push --force", "reset --hard", "clean -f", "branch -D"}
			for _, d := range dangerous {
				if strings.Contains(command, d) {
					return fmt.Sprintf("Blocked: '%s' is destructive. Use Bash if you really need this.", d)
				}
			}

			args := strings.Fields(command)
			cmd := exec.Command("git", args...)
			output, err := cmd.CombinedOutput()
			result := string(output)
			if err != nil {
				result += fmt.Sprintf("\nError: %v", err)
			}
			if len(result) > 50000 {
				result = result[:50000] + "\n... (truncated)"
			}
			if result == "" {
				return "(empty output)"
			}
			return strings.TrimRight(result, "\n")
		},
	}
}
