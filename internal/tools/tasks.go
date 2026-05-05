package tools

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

type taskEntry struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
}

var (
	taskStore   = make(map[string]*taskEntry)
	taskCounter int
	taskMu      sync.Mutex
)

func TaskCreateTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "TaskCreate",
			Description: "Create a tracked task for progress tracking.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"description": map[string]interface{}{"type": "string"},
					"status":      map[string]interface{}{"type": "string", "enum": []string{"pending", "in_progress", "completed", "blocked"}},
				},
				"required": []string{"description"},
			},
		},
		Execute: func(input map[string]interface{}) string {
			desc, _ := input["description"].(string)
			status, _ := input["status"].(string)
			if status == "" {
				status = "pending"
			}
			taskMu.Lock()
			taskCounter++
			id := fmt.Sprintf("task_%d", taskCounter)
			taskStore[id] = &taskEntry{
				ID:          id,
				Description: desc,
				Status:      status,
				Created:     time.Now().Format(time.RFC3339),
				Updated:     time.Now().Format(time.RFC3339),
			}
			taskMu.Unlock()
			return fmt.Sprintf("Created: %s — %s", id, desc)
		},
	}
}

func TaskUpdateTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "TaskUpdate",
			Description: "Update a task's status.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "string"},
					"status":  map[string]interface{}{"type": "string", "enum": []string{"pending", "in_progress", "completed", "blocked"}},
				},
				"required": []string{"task_id", "status"},
			},
		},
		Execute: func(input map[string]interface{}) string {
			id, _ := input["task_id"].(string)
			status, _ := input["status"].(string)
			taskMu.Lock()
			t, ok := taskStore[id]
			taskMu.Unlock()
			if !ok {
				return "Task not found: " + id
			}
			t.Status = status
			t.Updated = time.Now().Format(time.RFC3339)
			return fmt.Sprintf("Updated %s → %s", id, status)
		},
	}
}

func TaskGetTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "TaskGet",
			Description: "Get task info or list all tasks.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{"type": "string", "description": "Omit to list all"},
				},
			},
		},
		Execute: func(input map[string]interface{}) string {
			id, _ := input["task_id"].(string)
			taskMu.Lock()
			defer taskMu.Unlock()
			if id != "" {
				// Also check background tasks
				if bg, ok := _bg_commands_copy()[id]; ok {
					return fmt.Sprintf("Background: %s\nStatus: %s\nResult: %s",
						bg.command, bg.status, bg.result)
				}
				t, ok := taskStore[id]
				if !ok {
					return "Not found: " + id
				}
				data, _ := json.MarshalIndent(t, "", "  ")
				return string(data)
			}
			if len(taskStore) == 0 {
				return "No tasks"
			}
			var lines []string
			for _, t := range taskStore {
				icon := "•"
				switch t.Status {
				case "completed":
					icon = "✓"
				case "in_progress":
					icon = "→"
				case "blocked":
					icon = "✗"
				}
				lines = append(lines, fmt.Sprintf("%s %s [%s] %s", icon, t.ID, t.Status, t.Description))
			}
			return fmt.Sprintf("%d tasks:\n%s", len(taskStore), fmt.Sprintf("%s", lines))
		},
	}
}

type bgEntry struct {
	command string
	status  string
	result  string
}

func _bg_commands_copy() map[string]bgEntry {
	// Stub — background tasks are tracked in bash.go
	return nil
}
