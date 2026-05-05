package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

	// Try DuckDuckGo HTML
	results := searchDDG(query, maxResults)

	// Fallback: try Google scraping
	if len(results) == 0 {
		results = searchGoogle(query, maxResults)
	}

	// Fallback: SearXNG on brain if available
	if len(results) == 0 {
		results = searchSearXNG(query, maxResults)
	}

	if len(results) == 0 {
		return "No results for: " + query + "\nTip: use WebFetch with a direct URL instead."
	}
	return strings.Join(results, "\n\n")
}

func searchDDG(query string, max int) []string {
	req, _ := http.NewRequest("GET", "https://html.duckduckgo.com/html/?q="+url.QueryEscape(query), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	tagRe := regexp.MustCompile(`<[^>]+>`)
	var results []string

	// Pattern 1: result__a + result__snippet
	re1 := regexp.MustCompile(`(?s)<a[^>]*class="result__a"[^>]*href="([^"]+)"[^>]*>(.*?)</a>.*?<a[^>]*class="result__snippet"[^>]*>(.*?)</a>`)
	for _, m := range re1.FindAllStringSubmatch(html, max) {
		rawURL := m[1]
		title := strings.TrimSpace(tagRe.ReplaceAllString(m[2], ""))
		snippet := strings.TrimSpace(tagRe.ReplaceAllString(m[3], ""))
		if strings.Contains(rawURL, "uddg=") {
			if u, err := url.Parse(rawURL); err == nil {
				if uddg := u.Query().Get("uddg"); uddg != "" {
					rawURL, _ = url.QueryUnescape(uddg)
				}
			}
		}
		if title != "" {
			results = append(results, fmt.Sprintf("**%s**\n%s\n%s", title, rawURL, snippet))
		}
	}

	// Pattern 2: result-link + result__body (alternative DDG layout)
	if len(results) == 0 {
		re2 := regexp.MustCompile(`(?s)<a[^>]*class="[^"]*result-link[^"]*"[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
		for _, m := range re2.FindAllStringSubmatch(html, max) {
			rawURL := m[1]
			title := strings.TrimSpace(tagRe.ReplaceAllString(m[2], ""))
			if title != "" {
				results = append(results, fmt.Sprintf("**%s**\n%s", title, rawURL))
			}
		}
	}

	return results
}

func searchGoogle(query string, max int) []string {
	req, _ := http.NewRequest("GET", "https://www.google.com/search?q="+url.QueryEscape(query)+"&num="+fmt.Sprintf("%d", max), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	tagRe := regexp.MustCompile(`<[^>]+>`)
	var results []string

	// Google result pattern: <a href="/url?q=...">
	re := regexp.MustCompile(`<a[^>]*href="/url\?q=([^&"]+)[^"]*"[^>]*>(.*?)</a>`)
	for _, m := range re.FindAllStringSubmatch(html, max*2) {
		rawURL, _ := url.QueryUnescape(m[1])
		title := strings.TrimSpace(tagRe.ReplaceAllString(m[2], ""))
		if title == "" || strings.Contains(rawURL, "google.com") || strings.Contains(rawURL, "accounts.google") {
			continue
		}
		results = append(results, fmt.Sprintf("**%s**\n%s", title, rawURL))
		if len(results) >= max {
			break
		}
	}
	return results
}

func searchSearXNG(query string, max int) []string {
	// Try local SearXNG on brain
	req, _ := http.NewRequest("GET",
		"http://192.168.0.100:8888/search?q="+url.QueryEscape(query)+"&format=json&engines=google,duckduckgo,bing",
		nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}

	var data struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}

	var results []string
	for _, r := range data.Results {
		if len(results) >= max {
			break
		}
		results = append(results, fmt.Sprintf("**%s**\n%s\n%s", r.Title, r.URL, r.Content))
	}
	return results
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

// DownloadTool downloads a file from URL to local path.
func DownloadTool() Tool {
	return Tool{
		Def: provider.ToolDef{
			Name:        "Download",
			Description: "Download a file from URL to local disk. Use for downloading code, binaries, images, data files.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url":  map[string]interface{}{"type": "string", "description": "URL to download"},
					"path": map[string]interface{}{"type": "string", "description": "Local path to save (default: filename from URL in cwd)"},
				},
				"required": []string{"url"},
			},
		},
		Execute: executeDownload,
	}
}

func executeDownload(input map[string]interface{}) string {
	rawURL, _ := input["url"].(string)
	savePath, _ := input["path"].(string)

	if rawURL == "" {
		return "Error: url required"
	}

	// Default filename from URL
	if savePath == "" {
		parts := strings.Split(rawURL, "/")
		name := parts[len(parts)-1]
		if name == "" || len(name) > 100 {
			name = "download"
		}
		// Remove query params
		if idx := strings.Index(name, "?"); idx > 0 {
			name = name[:idx]
		}
		savePath = name
	}

	req, _ := http.NewRequest("GET", rawURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AuroraBot/1.0)")

	dlClient := &http.Client{Timeout: 5 * time.Minute}
	resp, err := dlClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Download error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Sprintf("HTTP %d for %s", resp.StatusCode, rawURL)
	}

	// Create parent dirs
	dir := filepath.Dir(savePath)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0755)
	}

	f, err := os.Create(savePath)
	if err != nil {
		return fmt.Sprintf("Create error: %v", err)
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Sprintf("Write error: %v", err)
	}

	return fmt.Sprintf("Downloaded %s → %s (%d bytes)", rawURL, savePath, written)
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

	// 1. Jina Reader — best quality, returns markdown
	if result := fetchViaJina(rawURL, maxBytes); result != "" {
		return result
	}
	// 2. Crawl4AI on brain — self-hosted fallback
	if result := fetchViaCrawl4AI(rawURL, maxBytes); result != "" {
		return result
	}
	// 3. Direct fetch — strip HTML manually
	return fetchDirect(rawURL, maxBytes)
}

func fetchViaJina(rawURL string, maxBytes int) string {
	jinaURL := "https://r.jina.ai/" + rawURL
	req, _ := http.NewRequest("GET", jinaURL, nil)
	req.Header.Set("Accept", "text/markdown")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp == nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return ""
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)))
	text := strings.TrimSpace(string(body))
	if len(text) < 50 {
		return ""
	}
	return text
}

func fetchViaCrawl4AI(rawURL string, maxBytes int) string {
	apiURL := "http://192.168.0.100:11235/crawl"
	payload := fmt.Sprintf(`{"urls":["%s"],"word_count_threshold":10}`, rawURL)
	req, _ := http.NewRequest("POST", apiURL, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp == nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return ""
	}
	defer resp.Body.Close()
	var data struct {
		Results []struct {
			Markdown string `json:"markdown"`
		} `json:"results"`
	}
	json.NewDecoder(resp.Body).Decode(&data)
	if len(data.Results) == 0 || data.Results[0].Markdown == "" {
		return ""
	}
	text := data.Results[0].Markdown
	if len(text) > maxBytes {
		text = text[:maxBytes]
	}
	return text
}

func fetchDirect(rawURL string, maxBytes int) string {
	req, _ := http.NewRequest("GET", rawURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Sprintf("Fetch error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes*2)))
	text := string(body)
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "json") {
		if len(text) > maxBytes {
			text = text[:maxBytes]
		}
		return text
	}
	if strings.Contains(ct, "html") {
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
