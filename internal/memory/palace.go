package memory

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Palace is the persistent per-user memory system.
type Palace struct {
	file   string
	userID string
	Data   PalaceData
}

type PalaceData struct {
	User        map[string]MemoryEntry `json:"user"`
	Projects    map[string]MemoryEntry `json:"projects"`
	Facts       []string               `json:"facts"`
	LastSummary string                 `json:"last_summary"`
}

type MemoryEntry struct {
	Value     string `json:"value"`
	Timestamp string `json:"ts"`
}

// New creates or loads a Memory Palace for a specific user.
func New(userToken string) *Palace {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".aurora", "memory")
	os.MkdirAll(dir, 0755)

	// Per-user memory file based on token hash
	filename := "palace.json" // default for owner
	if userToken != "" && userToken != "owner" {
		hash := md5.Sum([]byte(userToken))
		filename = fmt.Sprintf("palace_%x.json", hash[:4])
	}

	p := &Palace{
		file:   filepath.Join(dir, filename),
		userID: userToken,
	}
	p.load()
	return p
}

func (p *Palace) load() {
	data, err := os.ReadFile(p.file)
	if err != nil {
		p.Data = PalaceData{
			User:     make(map[string]MemoryEntry),
			Projects: make(map[string]MemoryEntry),
		}
		return
	}
	json.Unmarshal(data, &p.Data)
	if p.Data.User == nil {
		p.Data.User = make(map[string]MemoryEntry)
	}
	if p.Data.Projects == nil {
		p.Data.Projects = make(map[string]MemoryEntry)
	}
}

func (p *Palace) save() {
	data, _ := json.MarshalIndent(p.Data, "", "  ")
	os.WriteFile(p.file, data, 0644)
}

func (p *Palace) Remember(category, key, value string) {
	entry := MemoryEntry{Value: value, Timestamp: time.Now().Format(time.RFC3339)}
	switch category {
	case "user":
		p.Data.User[key] = entry
	case "projects", "project":
		p.Data.Projects[key] = entry
	default:
		p.Data.User[key] = entry
	}
	p.save()
}

func (p *Palace) AddFact(fact string) {
	for _, f := range p.Data.Facts {
		if f == fact {
			return
		}
	}
	p.Data.Facts = append(p.Data.Facts, fact)
	if len(p.Data.Facts) > 200 {
		p.Data.Facts = p.Data.Facts[len(p.Data.Facts)-200:]
	}
	p.save()
}

func (p *Palace) SaveSummary(summary string) {
	p.Data.LastSummary = summary
	p.save()
}

func (p *Palace) RecallAll() string {
	var parts []string
	if len(p.Data.User) > 0 {
		parts = append(parts, "## User")
		for k, v := range p.Data.User {
			parts = append(parts, fmt.Sprintf("- %s: %s", k, v.Value))
		}
	}
	if len(p.Data.Projects) > 0 {
		parts = append(parts, "\n## Projects")
		for k, v := range p.Data.Projects {
			parts = append(parts, fmt.Sprintf("- %s: %s", k, v.Value))
		}
	}
	if len(p.Data.Facts) > 0 {
		parts = append(parts, "\n## Facts")
		n := len(p.Data.Facts)
		start := 0
		if n > 30 {
			start = n - 30
		}
		for _, f := range p.Data.Facts[start:] {
			parts = append(parts, "- "+f)
		}
	}
	if p.Data.LastSummary != "" {
		parts = append(parts, "\n## Last Session\n"+p.Data.LastSummary)
	}
	return strings.Join(parts, "\n")
}

func (p *Palace) Count() int {
	return len(p.Data.User) + len(p.Data.Projects) + len(p.Data.Facts)
}
