package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectContext scans the working directory and returns context about the project.
func ProjectContext(workDir string) string {
	var parts []string

	// Detect project type
	checks := map[string]string{
		"go.mod":         "Go",
		"package.json":   "Node.js/JavaScript",
		"Cargo.toml":     "Rust",
		"pyproject.toml": "Python",
		"setup.py":       "Python",
		"requirements.txt": "Python",
		"pom.xml":        "Java/Maven",
		"build.gradle":   "Java/Gradle",
		"composer.json":  "PHP",
		"Gemfile":        "Ruby",
		"mix.exs":        "Elixir",
		"project.clj":    "Clojure",
		"deps.edn":       "Clojure",
		"CMakeLists.txt": "C/C++",
		"Makefile":       "Make",
		"Dockerfile":     "Docker",
		"docker-compose.yml": "Docker Compose",
		".git":           "Git repository",
	}

	var detected []string
	for file, lang := range checks {
		path := filepath.Join(workDir, file)
		if _, err := os.Stat(path); err == nil {
			detected = append(detected, lang)
		}
	}

	if len(detected) > 0 {
		parts = append(parts, "Project type: "+strings.Join(detected, ", "))
	}

	// Get git info
	if _, err := os.Stat(filepath.Join(workDir, ".git")); err == nil {
		// Try to get branch
		if data, err := os.ReadFile(filepath.Join(workDir, ".git", "HEAD")); err == nil {
			head := strings.TrimSpace(string(data))
			if strings.HasPrefix(head, "ref: refs/heads/") {
				branch := strings.TrimPrefix(head, "ref: refs/heads/")
				parts = append(parts, "Git branch: "+branch)
			}
		}
	}

	// Read CLAUDE.md or AURORA.md
	for _, name := range []string{"CLAUDE.md", "AURORA.md", "aurora.md"} {
		path := filepath.Join(workDir, name)
		if data, err := os.ReadFile(path); err == nil {
			content := string(data)
			if len(content) > 3000 {
				content = content[:3000] + "\n... (truncated)"
			}
			parts = append(parts, fmt.Sprintf("Project instructions (%s):\n%s", name, content))
			break
		}
	}

	// Count files
	fileCount := 0
	filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(workDir, path)
		// Skip hidden dirs and common junk
		if info.IsDir() {
			base := filepath.Base(rel)
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "__pycache__" ||
				base == "venv" || base == ".venv" || base == "dist" || base == "build" || base == "target" {
				return filepath.SkipDir
			}
			return nil
		}
		fileCount++
		if fileCount > 500 {
			return filepath.SkipAll
		}
		return nil
	})
	if fileCount > 0 {
		parts = append(parts, fmt.Sprintf("Files: ~%d", fileCount))
	}

	return strings.Join(parts, "\n")
}
