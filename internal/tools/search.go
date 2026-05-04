package tools

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

// GrepTool creates the Grep tool.
func GrepTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Grep",
			Description: "Search file contents using regex. Uses ripgrep if available.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{"type": "string", "description": "Regex pattern"},
					"path":    map[string]interface{}{"type": "string", "description": "Dir or file (default: cwd)"},
					"glob":    map[string]interface{}{"type": "string", "description": "File filter (e.g. '*.py')"},
					"output_mode": map[string]interface{}{
						"type": "string",
						"enum": []string{"content", "files_with_matches", "count"},
					},
					"context": map[string]interface{}{"type": "integer", "description": "Context lines"},
				},
				"required": []string{"pattern"},
			},
		},
		Execute: executeGrep,
	}
}

func executeGrep(input map[string]interface{}) string {
	pattern, _ := input["pattern"].(string)
	path, _ := input["path"].(string)
	fileGlob, _ := input["glob"].(string)
	mode, _ := input["output_mode"].(string)
	ctx := 0
	if c, ok := input["context"].(float64); ok {
		ctx = int(c)
	}

	if pattern == "" {
		return "Error: pattern required"
	}
	if path == "" {
		path = "."
	}
	if mode == "" {
		mode = "files_with_matches"
	}

	// Try ripgrep
	rg, err := exec.LookPath("rg")
	if err == nil {
		args := []string{"--no-heading", "--max-count", "200"}
		switch mode {
		case "files_with_matches":
			args = append(args, "-l")
		case "count":
			args = append(args, "-c")
		default:
			args = append(args, "-n")
		}
		if ctx > 0 {
			args = append(args, "-C", fmt.Sprintf("%d", ctx))
		}
		if fileGlob != "" {
			args = append(args, "--glob", fileGlob)
		}
		args = append(args, pattern, path)

		cmd := exec.Command(rg, args...)
		output, _ := cmd.CombinedOutput()
		result := strings.TrimRight(string(output), "\n")
		if result == "" {
			return "(no matches)"
		}
		if len(result) > 50000 {
			result = result[:50000] + "\n... (truncated)"
		}
		return result
	}

	return "(ripgrep not found, install rg for grep support)"
}

// GlobTool creates the Glob tool.
func GlobTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Glob",
			Description: "Find files matching a glob pattern (e.g. '**/*.py').",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{"type": "string", "description": "Glob pattern"},
					"path":    map[string]interface{}{"type": "string", "description": "Base dir (default: cwd)"},
				},
				"required": []string{"pattern"},
			},
		},
		Execute: executeGlob,
	}
}

func executeGlob(input map[string]interface{}) string {
	pattern, _ := input["pattern"].(string)
	path, _ := input["path"].(string)
	if pattern == "" {
		return "Error: pattern required"
	}
	if path == "" {
		path = "."
	}

	fullPattern := filepath.Join(path, pattern)
	matches, err := filepath.Glob(fullPattern)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	if len(matches) == 0 {
		return "(no matches)"
	}
	if len(matches) > 200 {
		matches = matches[:200]
	}
	return strings.Join(matches, "\n")
}
