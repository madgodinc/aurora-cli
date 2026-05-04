package tools

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/madgodinc/aurora-cli/internal/provider"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// WebSearchTool creates the web search tool.
func WebSearchTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "WebSearch",
			Description: "Search the web using DuckDuckGo. Returns titles, URLs, snippets.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query":       map[string]interface{}{"type": "string", "description": "Search query"},
					"max_results": map[string]interface{}{"type": "integer", "description": "Max results (default 8)"},
				},
				"required": []string{"query"},
			},
		},
		Execute: executeWebSearch,
	}
}

func executeWebSearch(input map[string]interface{}) string {
	query, _ := input["query"].(string)
	maxResults := 8
	if m, ok := input["max_results"].(float64); ok && m > 0 {
		maxResults = int(m)
	}

	if query == "" {
		return "Error: query required"
	}

	req, _ := http.NewRequest("GET", "https://html.duckduckgo.com/html/?q="+url.QueryEscape(query), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Search error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Parse results
	re := regexp.MustCompile(`<a rel="nofollow" class="result__a" href="([^"]+)"[^>]*>(.*?)</a>.*?<a class="result__snippet"[^>]*>(.*?)</a>`)
	matches := re.FindAllStringSubmatch(html, maxResults)

	tagRe := regexp.MustCompile(`<[^>]+>`)
	var results []string
	for _, m := range matches {
		rawURL := m[1]
		title := tagRe.ReplaceAllString(m[2], "")
		snippet := tagRe.ReplaceAllString(m[3], "")

		// Decode DDG redirect
		if strings.Contains(rawURL, "uddg=") {
			if u, err := url.Parse(rawURL); err == nil {
				if uddg := u.Query().Get("uddg"); uddg != "" {
					rawURL = uddg
				}
			}
		}

		results = append(results, fmt.Sprintf("**%s**\n%s\n%s", strings.TrimSpace(title), rawURL, strings.TrimSpace(snippet)))
	}

	if len(results) == 0 {
		return "No results for: " + query
	}
	return strings.Join(results, "\n\n")
}

// WebFetchTool creates the URL fetch tool.
func WebFetchTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "WebFetch",
			Description: "Fetch a URL and return content (HTML→text, JSON). Max 100KB.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url":       map[string]interface{}{"type": "string", "description": "URL to fetch"},
					"max_bytes": map[string]interface{}{"type": "integer", "description": "Max response size (default 100000)"},
				},
				"required": []string{"url"},
			},
		},
		Execute: executeWebFetch,
	}
}

func executeWebFetch(input map[string]interface{}) string {
	rawURL, _ := input["url"].(string)
	maxBytes := 100000
	if m, ok := input["max_bytes"].(float64); ok && m > 0 {
		maxBytes = int(m)
	}

	if rawURL == "" {
		return "Error: url required"
	}

	req, _ := http.NewRequest("GET", rawURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AuroraBot/1.0)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Fetch error: %v", err)
	}
	defer resp.Body.Close()

	// Read limited
	limited := io.LimitReader(resp.Body, int64(maxBytes*2))
	body, _ := io.ReadAll(limited)
	text := string(body)

	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		if len(text) > maxBytes {
			text = text[:maxBytes]
		}
		return text
	}

	if strings.Contains(ct, "html") {
		// Strip scripts, styles, tags
		scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
		styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
		tagRe := regexp.MustCompile(`<[^>]+>`)
		spaceRe := regexp.MustCompile(`\s+`)

		text = scriptRe.ReplaceAllString(text, "")
		text = styleRe.ReplaceAllString(text, "")
		text = tagRe.ReplaceAllString(text, " ")
		text = spaceRe.ReplaceAllString(text, " ")
		text = strings.TrimSpace(text)
	}

	if len(text) > maxBytes {
		text = text[:maxBytes]
	}
	return text
}
