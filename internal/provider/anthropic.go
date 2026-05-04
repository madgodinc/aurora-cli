package provider

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Config holds provider configuration.
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

// Message represents a conversation message.
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ContentBlock
}

// ContentBlock is a piece of message content.
type ContentBlock struct {
	Type      string      `json:"type"`
	Text      string      `json:"text,omitempty"`
	ID        string      `json:"id,omitempty"`
	Name      string      `json:"name,omitempty"`
	Input     interface{} `json:"input,omitempty"`
	ToolUseID string      `json:"tool_use_id,omitempty"` // for tool_result
	Content   interface{} `json:"content,omitempty"`      // for tool_result content
}

// ToolDef defines a tool for the API.
type ToolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"input_schema"`
}

// StreamEvent represents a parsed SSE event from the API.
type StreamEvent struct {
	Type string // "text", "tool_use_start", "tool_input_delta", "tool_use_stop", "message_start", "message_delta", "message_stop", "error"

	// Text delta
	Text string

	// Tool use
	ToolID    string
	ToolName  string
	ToolJSON  string // partial JSON for input

	// Message-level
	StopReason  string
	InputTokens int
	OutputTokens int
}

// Request is a messages API request.
type Request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []Message `json:"messages"`
	Tools     []ToolDef `json:"tools,omitempty"`
	Stream    bool      `json:"stream"`
}

// Client handles communication with the Anthropic-compatible API.
type Client struct {
	config Config
	http   *http.Client
}

// NewClient creates a new API client.
func NewClient(cfg Config) *Client {
	return &Client{
		config: cfg,
		http:   &http.Client{},
	}
}

// Stream sends a request and returns a channel of StreamEvents.
func (c *Client) Stream(req Request) (<-chan StreamEvent, error) {
	req.Stream = true
	if req.Model == "" {
		req.Model = c.config.Model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.config.BaseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		errMsg := string(b)
		if len(errMsg) > 500 {
			errMsg = errMsg[:500]
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, errMsg)
	}

	ch := make(chan StreamEvent, 100)
	go c.parseSSE(resp.Body, ch)
	return ch, nil
}

func (c *Client) parseSSE(body io.ReadCloser, ch chan<- StreamEvent) {
	defer close(ch)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var eventType string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		dataStr := strings.TrimPrefix(line, "data: ")
		if dataStr == "[DONE]" {
			break
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
			continue
		}

		_ = eventType // used for context
		dtype, _ := data["type"].(string)

		switch dtype {
		case "message_start":
			msg, _ := data["message"].(map[string]interface{})
			usage, _ := msg["usage"].(map[string]interface{})
			inputTok := int(getFloat(usage, "input_tokens"))
			ch <- StreamEvent{Type: "message_start", InputTokens: inputTok}

		case "content_block_start":
			block, _ := data["content_block"].(map[string]interface{})
			btype, _ := block["type"].(string)
			if btype == "text" {
				ch <- StreamEvent{Type: "text_start"}
			} else if btype == "tool_use" {
				name, _ := block["name"].(string)
				id, _ := block["id"].(string)
				ch <- StreamEvent{Type: "tool_use_start", ToolName: name, ToolID: id}
			}

		case "content_block_delta":
			delta, _ := data["delta"].(map[string]interface{})
			dtype, _ := delta["type"].(string)
			if dtype == "text_delta" {
				text, _ := delta["text"].(string)
				ch <- StreamEvent{Type: "text", Text: text}
			} else if dtype == "input_json_delta" {
				pj, _ := delta["partial_json"].(string)
				ch <- StreamEvent{Type: "tool_input_delta", ToolJSON: pj}
			}

		case "content_block_stop":
			ch <- StreamEvent{Type: "block_stop"}

		case "message_delta":
			delta, _ := data["delta"].(map[string]interface{})
			stopReason, _ := delta["stop_reason"].(string)
			usage, _ := data["usage"].(map[string]interface{})
			outputTok := int(getFloat(usage, "output_tokens"))
			ch <- StreamEvent{Type: "message_delta", StopReason: stopReason, OutputTokens: outputTok}

		case "message_stop":
			ch <- StreamEvent{Type: "message_stop"}
		}
	}
}

func getFloat(m map[string]interface{}, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key].(float64)
	if !ok {
		return 0
	}
	return v
}
