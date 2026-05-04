package session

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Session holds conversation messages and metadata.
type Session struct {
	ID           string                   `json:"id"`
	WorkDir      string                   `json:"work_dir"`
	Messages     []map[string]interface{} `json:"messages"`
	InputTokens  int                      `json:"input_tokens"`
	OutputTokens int                      `json:"output_tokens"`
	Updated      time.Time                `json:"updated"`
}

var sessionsDir string

func init() {
	home, _ := os.UserHomeDir()
	sessionsDir = filepath.Join(home, ".aurora", "sessions")
	os.MkdirAll(sessionsDir, 0755)
}

// NewID creates a session ID based on working directory.
func NewID(workDir string) string {
	h := md5.Sum([]byte(workDir))
	return fmt.Sprintf("s_%d_%x", time.Now().Unix(), h[:3])
}

// Save persists session to disk.
func Save(s *Session) error {
	s.Updated = time.Now()
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	path := filepath.Join(sessionsDir, s.ID+".json")
	return os.WriteFile(path, data, 0644)
}

// Load reads a session by ID.
func Load(id string) (*Session, error) {
	path := filepath.Join(sessionsDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// FindLatest finds the most recent session for a work directory.
func FindLatest(workDir string) *Session {
	entries, _ := os.ReadDir(sessionsDir)
	var best *Session
	var bestTime time.Time

	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sessionsDir, e.Name()))
		if err != nil {
			continue
		}
		var s Session
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		if s.WorkDir == workDir && s.Updated.After(bestTime) {
			bestTime = s.Updated
			best = &s
		}
	}
	return best
}

// List returns all sessions, most recent first.
func List() []Session {
	entries, _ := os.ReadDir(sessionsDir)
	var sessions []Session
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(sessionsDir, e.Name()))
		var s Session
		if json.Unmarshal(data, &s) == nil {
			sessions = append(sessions, s)
		}
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Updated.After(sessions[j].Updated)
	})
	return sessions
}
