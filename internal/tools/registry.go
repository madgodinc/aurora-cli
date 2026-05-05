package tools

import (
	"github.com/madgodinc/aurora-cli/internal/provider"
)

// Tool represents an executable tool.
type Tool struct {
	Def     provider.ToolDef
	Execute func(input map[string]interface{}) string
}

// Registry returns all available tools for the given access level.
func Registry(hasSSH bool) []Tool {
	t := []Tool{
		// Core
		BashTool(),
		ReadTool(),
		WriteTool(),
		EditTool(),
		GrepTool(),
		GlobTool(),
		// Web
		WebSearchTool(),
		WebFetchTool(),
		// Dev
		GitTool(),
		// System
		ClipboardTool(),
		ProcessTool(),
		// Code
		MultiEditTool(),
		RunTestTool(),
		TreeTool(),
		DiffTool(),
		// Agent
		AgentTool(),
		// Notebook
		NotebookEditTool(),
		// Tasks
		TaskCreateTool(),
		TaskUpdateTool(),
		TaskGetTool(),
		// Memory
		RememberTool(),
	}

	// Server tools — only with SSH access
	if hasSSH {
		t = append(t,
			RemoteShellTool(),
			DockerTool(),
			ServerDashboardTool(),
			ModelSwitchTool(),
		)
	}

	return t
}

// ToolDefs returns just the API definitions.
func ToolDefs(hasSSH bool) []provider.ToolDef {
	tools := Registry(hasSSH)
	defs := make([]provider.ToolDef, len(tools))
	for i, t := range tools {
		defs[i] = t.Def
	}
	return defs
}

// FindTool looks up a tool by name.
func FindTool(name string, hasSSH bool) *Tool {
	for _, t := range Registry(hasSSH) {
		if t.Def.Name == name {
			return &t
		}
	}
	return nil
}

// ToolCount returns the number of available tools.
func ToolCount(hasSSH bool) int {
	return len(Registry(hasSSH))
}
