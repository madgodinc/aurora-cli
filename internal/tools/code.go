package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

// MultiEditTool does batch edits on a file.
func MultiEditTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "MultiEdit",
			Description: "Apply multiple find-replace edits to a file in one call. Each edit is {old, new}.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{"type": "string"},
					"edits": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"old": map[string]interface{}{"type": "string"},
								"new": map[string]interface{}{"type": "string"},
							},
						},
					},
				},
				"required": []string{"file_path", "edits"},
			},
		},
		Execute: func(input map[string]interface{}) string {
			fp, _ := input["file_path"].(string)
			editsRaw, _ := input["edits"].([]interface{})
			if fp == "" || len(editsRaw) == 0 {
				return "Error: file_path and edits required"
			}
			data, err := os.ReadFile(fp)
			if err != nil {
				return fmt.Sprintf("Error: %v", err)
			}
			content := string(data)
			applied := 0
			for _, e := range editsRaw {
				em, ok := e.(map[string]interface{})
				if !ok {
					continue
				}
				old, _ := em["old"].(string)
				new_, _ := em["new"].(string)
				if old == "" {
					continue
				}
				if strings.Contains(content, old) {
					content = strings.Replace(content, old, new_, 1)
					applied++
				}
			}
			if applied == 0 {
				return "No edits matched"
			}
			os.WriteFile(fp, []byte(content), 0644)
			return fmt.Sprintf("Applied %d/%d edits to %s", applied, len(editsRaw), fp)
		},
	}
}

// RunTestTool runs test commands and parses output.
func RunTestTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "RunTest",
			Description: "Run project tests. Auto-detects: go test, npm test, pytest, cargo test. Or specify custom command.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "Custom test command. Leave empty for auto-detect.",
					},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory to run tests in (default: cwd)",
					},
				},
			},
		},
		Execute: func(input map[string]interface{}) string {
			command, _ := input["command"].(string)
			path, _ := input["path"].(string)
			if path == "" {
				path = "."
			}

			if command == "" {
				// Auto-detect
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					command = "go test ./... -v -count=1 2>&1 | tail -50"
				} else if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
					command = "npm test 2>&1 | tail -50"
				} else if _, err := os.Stat(filepath.Join(path, "pytest.ini")); err == nil {
					command = "pytest -v 2>&1 | tail -50"
				} else if _, err := os.Stat(filepath.Join(path, "Cargo.toml")); err == nil {
					command = "cargo test 2>&1 | tail -50"
				} else {
					return "No test framework detected. Use command parameter."
				}
			}

			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				bash := `C:\Program Files\Git\bin\bash.exe`
				cmd = exec.Command(bash, "-c", "cd '"+path+"' && "+command)
			} else {
				cmd = exec.Command("bash", "-c", "cd '"+path+"' && "+command)
			}
			trackChild(cmd)
			defer untrackChild(cmd)

			output, err := cmd.CombinedOutput()
			result := string(output)
			if err != nil {
				result += fmt.Sprintf("\nExit: %v", err)
			}
			if len(result) > 30000 {
				result = result[:30000] + "\n... (truncated)"
			}
			return result
		},
	}
}

// TreeTool shows directory tree.
func TreeTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Tree",
			Description: "Show directory tree structure. Useful for understanding project layout.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Directory path (default: cwd)",
					},
					"depth": map[string]interface{}{
						"type":        "integer",
						"description": "Max depth (default: 3)",
					},
				},
			},
		},
		Execute: func(input map[string]interface{}) string {
			path, _ := input["path"].(string)
			if path == "" {
				path = "."
			}
			maxDepth := 3
			if d, ok := input["depth"].(float64); ok && d > 0 {
				maxDepth = int(d)
			}

			var sb strings.Builder
			sb.WriteString(path + "/\n")
			buildTree(&sb, path, "", 0, maxDepth)
			result := sb.String()
			if len(result) > 20000 {
				result = result[:20000] + "\n... (truncated)"
			}
			return result
		},
	}
}

func buildTree(sb *strings.Builder, dir, prefix string, depth, maxDepth int) {
	if depth >= maxDepth {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Filter
	var filtered []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		skip := map[string]bool{
			"node_modules": true, "__pycache__": true, "venv": true,
			".venv": true, "dist": true, "build": true, "target": true,
			".git": true, ".idea": true, ".vscode": true,
		}
		if skip[name] {
			continue
		}
		filtered = append(filtered, e)
	}

	for i, e := range filtered {
		isLast := i == len(filtered)-1
		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		fmt.Fprintf(sb, "%s%s%s\n", prefix, connector, name)

		if e.IsDir() {
			buildTree(sb, filepath.Join(dir, e.Name()), prefix+childPrefix, depth+1, maxDepth)
		}
	}
}

// DiffTool shows diff between files or git changes.
func DiffTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Diff",
			Description: "Show differences. Modes: 'git' (unstaged changes), 'staged' (staged changes), 'files' (two files).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"mode": map[string]interface{}{
						"type": "string",
						"enum": []string{"git", "staged", "files"},
					},
					"file_a": map[string]interface{}{"type": "string", "description": "First file (for files mode)"},
					"file_b": map[string]interface{}{"type": "string", "description": "Second file (for files mode)"},
				},
				"required": []string{"mode"},
			},
		},
		Execute: func(input map[string]interface{}) string {
			mode, _ := input["mode"].(string)
			switch mode {
			case "git":
				cmd := exec.Command("git", "diff")
				out, _ := cmd.CombinedOutput()
				if len(out) == 0 {
					return "(no unstaged changes)"
				}
				return string(out)
			case "staged":
				cmd := exec.Command("git", "diff", "--cached")
				out, _ := cmd.CombinedOutput()
				if len(out) == 0 {
					return "(no staged changes)"
				}
				return string(out)
			case "files":
				a, _ := input["file_a"].(string)
				b, _ := input["file_b"].(string)
				if a == "" || b == "" {
					return "Error: file_a and file_b required"
				}
				cmd := exec.Command("diff", "-u", a, b)
				out, _ := cmd.CombinedOutput()
				return string(out)
			}
			return "Error: mode must be git, staged, or files"
		},
	}
}
