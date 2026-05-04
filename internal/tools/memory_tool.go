package tools

import (
	"strings"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

// MemoryCallback is set by the agent to connect tool to memory palace.
var MemoryCallback func(action, category, key, value string) string

// RememberTool allows Aurora to autonomously save memories.
func RememberTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name: "Remember",
			Description: `Save important information to Memory Palace for future conversations.
Use this AUTOMATICALLY when you learn something worth remembering:
- User preferences, name, role, expertise
- Project details, architecture decisions, tech stack
- Server configs, credentials, access patterns
- Recurring tasks, workflows, habits
- Bug fixes, solutions that were hard to find
- Any fact that would save time if recalled later

Categories: user, project, fact.
Do NOT ask permission — just save when relevant.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"save", "recall", "list"},
						"description": "save=store new memory, recall=search memories, list=show all",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"user", "project", "fact"},
						"description": "Memory category",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "Memory key/name (for save)",
					},
					"value": map[string]interface{}{
						"type":        "string",
						"description": "What to remember (for save)",
					},
				},
				"required": []string{"action"},
			},
		},
		Execute: executeRemember,
	}
}

func executeRemember(input map[string]interface{}) string {
	action, _ := input["action"].(string)
	category, _ := input["category"].(string)
	key, _ := input["key"].(string)
	value, _ := input["value"].(string)

	if MemoryCallback == nil {
		return "Memory system not initialized"
	}

	switch action {
	case "save":
		if key == "" || value == "" {
			return "Error: key and value required for save"
		}
		if category == "" {
			category = "fact"
		}
		return MemoryCallback("save", category, key, value)

	case "recall":
		return MemoryCallback("recall", "", "", "")

	case "list":
		return MemoryCallback("list", "", "", "")

	default:
		return "Error: action must be save, recall, or list"
	}
}

// AutoMemoryPrompt returns instructions for Aurora to auto-save memories.
func AutoMemoryPrompt() string {
	return strings.TrimSpace(`
## Auto-Memory Rules
You have a Remember tool. Use it PROACTIVELY — don't ask permission.
After answering, if you learned any of these, CALL Remember immediately:
- User's name, role, preferences, expertise level
- Project name, tech stack, architecture
- Server details, configs, access patterns
- Solutions to problems (for future reference)
- User's communication style preferences
- Important decisions or constraints

Example: if user says "I'm a Go developer working on a game engine",
call Remember(save, user, role, "Go developer, working on game engine").
This data persists across sessions and helps you be more helpful next time.
`)
}
