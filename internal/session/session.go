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

// Session represents a persistent conversation session.
type Session struct {
	ID           string    `json:"id"`
	WorkDir      string    `json:"work_dir"`
	Messages     int       `json:"messages"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Updated      time.Time `json:"updated"`
}

var sessionsDir string

func init() {
	home, _ := os.UserHomeDir()
	sessionsDir = filepath.Join(home, ".aurora", "sessions")
	os.MkdirAll(sessionsDir, 0755)
}

// GenerateID creates a session ID based on working directory and timestamp.
func GenerateID(workDir string) string {
	h := md5.Sum([]byte(workDir))
	return fmt.Sprintf("s_%d_%x", time.Now().Unix(), h[:3])
}

// Save persists session metadata.
func Save(s Session) error {
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
func FindLatest(workDir string) string {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return ""
	}

	var best string
	var bestTime time.Time

	for _, e := range entries {
		if !e.Type().IsRegular() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(sessionsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var s Session
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		if s.WorkDir == workDir && s.Updated.After(bestTime) {
			bestTime = s.Updated
			best = s.ID
		}
	}
	return best
}

// List returns all sessions, sorted by most recent.
func List() []Session {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil
	}

	var sessions []Session
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(sessionsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var s Session
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Updated.After(sessions[j].Updated)
	})
	return sessions
}
