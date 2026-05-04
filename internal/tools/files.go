package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

// ReadTool creates the Read tool.
func ReadTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Read",
			Description: "Read a file. Returns numbered lines. Use offset/limit for large files.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{"type": "string", "description": "Absolute path to file"},
					"offset":    map[string]interface{}{"type": "integer", "description": "Start line (1-based, default 1)"},
					"limit":     map[string]interface{}{"type": "integer", "description": "Max lines (default 2000)"},
				},
				"required": []string{"file_path"},
			},
		},
		Execute: executeRead,
	}
}

func executeRead(input map[string]interface{}) string {
	fp, _ := input["file_path"].(string)
	if fp == "" {
		return "Error: file_path required"
	}

	offset := 1
	if o, ok := input["offset"].(float64); ok && o > 0 {
		offset = int(o)
	}
	limit := 2000
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	start := offset - 1
	if start < 0 {
		start = 0
	}
	end := start + limit
	if end > len(lines) {
		end = len(lines)
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		fmt.Fprintf(&sb, "%d\t%s\n", i+1, lines[i])
	}
	if end < len(lines) {
		fmt.Fprintf(&sb, "\n... (%d more lines)", len(lines)-end)
	}

	result := sb.String()
	if len(result) > 100000 {
		result = result[:100000] + "\n... (truncated)"
	}
	if result == "" {
		return "(empty file)"
	}
	return result
}

// WriteTool creates the Write tool.
func WriteTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Write",
			Description: "Write content to a file. Creates parent dirs. Overwrites existing.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{"type": "string", "description": "Absolute path"},
					"content":   map[string]interface{}{"type": "string", "description": "File content"},
				},
				"required": []string{"file_path", "content"},
			},
		},
		Execute: executeWrite,
	}
}

func executeWrite(input map[string]interface{}) string {
	fp, _ := input["file_path"].(string)
	content, _ := input["content"].(string)
	if fp == "" {
		return "Error: file_path required"
	}

	dir := filepath.Dir(fp)
	if dir != "" {
		os.MkdirAll(dir, 0755)
	}

	if err := os.WriteFile(fp, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Written %d chars to %s", len(content), fp)
}

// EditTool creates the Edit tool.
func EditTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Edit",
			Description: "Edit file by exact string replacement. old_string must be unique unless replace_all=true.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path":   map[string]interface{}{"type": "string", "description": "Absolute path"},
					"old_string":  map[string]interface{}{"type": "string", "description": "Exact text to find"},
					"new_string":  map[string]interface{}{"type": "string", "description": "Replacement"},
					"replace_all": map[string]interface{}{"type": "boolean", "description": "Replace all occurrences"},
				},
				"required": []string{"file_path", "old_string", "new_string"},
			},
		},
		Execute: executeEdit,
	}
}

func executeEdit(input map[string]interface{}) string {
	fp, _ := input["file_path"].(string)
	old, _ := input["old_string"].(string)
	new_, _ := input["new_string"].(string)
	replaceAll, _ := input["replace_all"].(bool)

	if fp == "" || old == "" {
		return "Error: file_path and old_string required"
	}

	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	content := string(data)
	count := strings.Count(content, old)
	if count == 0 {
		return fmt.Sprintf("old_string not found in %s", fp)
	}
	if count > 1 && !replaceAll {
		return fmt.Sprintf("old_string found %d times — not unique. Add context or use replace_all.", count)
	}

	if replaceAll {
		content = strings.ReplaceAll(content, old, new_)
	} else {
		content = strings.Replace(content, old, new_, 1)
	}

	if err := os.WriteFile(fp, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error writing: %v", err)
	}
	return fmt.Sprintf("Edited %s (%d replacement(s))", fp, count)
}
