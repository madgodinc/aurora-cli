package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const updateCheckInterval = 6 * time.Hour

// CheckForUpdate checks GitHub for a newer version and offers to update.
func CheckForUpdate(currentVersion string) {
	// Don't check too often
	home, _ := os.UserHomeDir()
	stampFile := filepath.Join(home, ".aurora", "last_update_check")
	if data, err := os.ReadFile(stampFile); err == nil {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data))); err == nil {
			if time.Since(t) < updateCheckInterval {
				return
			}
		}
	}

	// Save check timestamp
	os.MkdirAll(filepath.Dir(stampFile), 0755)
	os.WriteFile(stampFile, []byte(time.Now().Format(time.RFC3339)), 0644)

	// Get remote version from go.mod or main.go
	resp, err := http.Get("https://raw.githubusercontent.com/madgodinc/aurora-cli/main/main.go")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}
	body, _ := io.ReadAll(resp.Body)
	content := string(body)

	// Extract version from: const version = "0.2.0"
	idx := strings.Index(content, `const version = "`)
	if idx < 0 {
		return
	}
	start := idx + len(`const version = "`)
	end := strings.Index(content[start:], `"`)
	if end < 0 {
		return
	}
	remoteVersion := content[start : start+end]

	if remoteVersion == currentVersion {
		return
	}

	// Compare versions (simple: if different, offer update)
	fmt.Printf("\n  New version available: %s -> %s\n", currentVersion, remoteVersion)
	fmt.Print("  Update now? [Y/n]: ")
	var yn string
	fmt.Scanln(&yn)
	if strings.HasPrefix(strings.ToLower(yn), "n") {
		return
	}

	doUpdate()
}

func doUpdate() {
	fmt.Println("  Updating...")

	// Check Go
	goPath := "go"
	if _, err := exec.LookPath("go"); err != nil {
		goPath = "/c/go/bin/go"
		if _, err := os.Stat(goPath + ".exe"); err != nil {
			fmt.Println("  Error: Go not found. Run install.sh again.")
			return
		}
	}

	// Clone latest
	tmpDir, _ := os.MkdirTemp("", "aurora-update-*")
	defer os.RemoveAll(tmpDir)

	cmd := exec.Command("git", "clone", "--depth", "1",
		"https://github.com/madgodinc/aurora-cli.git",
		filepath.Join(tmpDir, "aurora-cli"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("  Clone failed: %v\n", err)
		return
	}

	// Build
	buildDir := filepath.Join(tmpDir, "aurora-cli")
	buildCmd := exec.Command(goPath, "build", "-ldflags=-s -w", "-o", "aurora_new.exe", ".")
	buildCmd.Dir = buildDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Printf("  Build failed: %v\n", err)
		return
	}

	// Replace binary
	home, _ := os.UserHomeDir()
	binPath := filepath.Join(home, "bin", "aurora.exe")
	newBin := filepath.Join(buildDir, "aurora_new.exe")

	// On Windows, can't replace running exe. Rename old, copy new.
	oldBin := binPath + ".old"
	os.Remove(oldBin)
	os.Rename(binPath, oldBin)
	data, err := os.ReadFile(newBin)
	if err != nil {
		// Restore old
		os.Rename(oldBin, binPath)
		fmt.Printf("  Read failed: %v\n", err)
		return
	}
	if err := os.WriteFile(binPath, data, 0755); err != nil {
		os.Rename(oldBin, binPath)
		fmt.Printf("  Write failed: %v\n", err)
		return
	}

	// Also update the build directory copy
	exePath, _ := os.Executable()
	if exePath != binPath {
		os.WriteFile(exePath+".old", nil, 0644) // mark for cleanup
		// Can't replace self, will take effect next launch
	}

	fmt.Println("  ✓ Updated! Restart aurora to use new version.")

	// Cleanup old
	go func() {
		time.Sleep(time.Second)
		os.Remove(oldBin)
	}()
}

// LatestReleaseInfo fetches latest release info from GitHub.
func LatestReleaseInfo() string {
	resp, err := http.Get("https://api.github.com/repos/madgodinc/aurora-cli/releases/latest")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var data struct {
		TagName string `json:"tag_name"`
		Body    string `json:"body"`
	}
	json.Unmarshal(body, &data)
	return data.TagName
}
