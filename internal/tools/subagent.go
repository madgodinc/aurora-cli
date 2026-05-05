package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

var (
	subagentProxyURL string
	subagentAPIKey   string
	subagentModel    string
	subagentMu       sync.Mutex
)

func SetSubagentConfig(proxyURL, apiKey, model string) {
	subagentMu.Lock()
	subagentProxyURL = proxyURL
	subagentAPIKey = apiKey
	subagentModel = model
	subagentMu.Unlock()
}

func AgentTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name: "Agent",
			Description: "Spawn a sub-agent for complex tasks. Runs independently with its own tool loop. Use for parallel research, code exploration, multi-step tasks.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"description": map[string]interface{}{"type": "string", "description": "Short task description (3-5 words)"},
					"prompt":      map[string]interface{}{"type": "string", "description": "Detailed task for the sub-agent"},
				},
				"required": []string{"description", "prompt"},
			},
		},
		Execute: executeSubagent,
	}
}

func executeSubagent(input map[string]interface{}) string {
	description, _ := input["description"].(string)
	prompt, _ := input["prompt"].(string)
	if prompt == "" {
		return "Error: prompt required"
	}

	subagentMu.Lock()
	proxyURL := subagentProxyURL
	apiKey := subagentAPIKey
	model := subagentModel
	subagentMu.Unlock()

	if proxyURL == "" {
		return "Error: sub-agent not configured"
	}

	system := fmt.Sprintf("You are a sub-agent of Aurora. Task: %s\nWorking directory: %s\nBe thorough but concise in your final answer.\nDate: %s",
		description, getWorkDir(), time.Now().Format("2006-01-02 15:04"))

	messages := []map[string]interface{}{
		{"role": "user", "content": prompt},
	}

	// Get tool defs
	hasSSH := SSHHost != ""
	toolDefs := ToolDefs(hasSSH)

	var finalText string

	for round := 0; round < 8; round++ {
		body := map[string]interface{}{
			"model":      model,
			"max_tokens": 4096,
			"system":     system,
			"messages":   messages,
			"tools":      toolDefs,
			"stream":     false,
		}

		bodyBytes, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", proxyURL+"/v1/messages", strings.NewReader(string(bodyBytes)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		client := &http.Client{Timeout: 3 * time.Minute}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Sprintf("Sub-agent error: %v", err)
		}

		var data map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()

		content, _ := data["content"].([]interface{})
		stopReason, _ := data["stop_reason"].(string)
		messages = append(messages, map[string]interface{}{"role": "assistant", "content": content})

		// Collect text
		for _, block := range content {
			bm, ok := block.(map[string]interface{})
			if !ok {
				continue
			}
			if bm["type"] == "text" {
				text, _ := bm["text"].(string)
				finalText += text
			}
		}

		// Check for tool use
		var toolUses []map[string]interface{}
		for _, block := range content {
			bm, ok := block.(map[string]interface{})
			if !ok {
				continue
			}
			if bm["type"] == "tool_use" {
				toolUses = append(toolUses, bm)
			}
		}

		if len(toolUses) == 0 || stopReason == "end_turn" {
			break
		}

		// Execute tools
		var results []map[string]interface{}
		for _, tu := range toolUses {
			name, _ := tu["name"].(string)
			id, _ := tu["id"].(string)
			inp, _ := tu["input"].(map[string]interface{})
			t := FindTool(name, hasSSH)
			var result string
			if t != nil {
				result = t.Execute(inp)
			} else {
				result = "Unknown tool: " + name
			}
			if len(result) > 15000 {
				result = result[:15000] + "\n... (truncated)"
			}
			results = append(results, map[string]interface{}{
				"type":        "tool_result",
				"tool_use_id": id,
				"content":     result,
			})
		}
		messages = append(messages, map[string]interface{}{"role": "user", "content": results})

		if stopReason == "end_turn" {
			break
		}
	}

	if finalText == "" {
		return "(sub-agent produced no output)"
	}
	return finalText
}
