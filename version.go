package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Version variables - injected by ldflags during build
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

const (
	githubAPIURL     = "https://api.github.com/repos/barberob/db-file-to-container-importer/releases/latest"
	updateCheckFile  = "last_update_check"
	checkInterval    = 24 * time.Hour
)

type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
}

type UpdateCheck struct {
	LastCheck   time.Time `json:"lastCheck"`
	LastVersion string    `json:"lastVersion"`
}

var (
	warningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFA500")).
		Bold(true)
	
	infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666"))
)

// checkForUpdate checks for new version asynchronously (non-blocking)
func checkForUpdateAsync() {
	go func() {
		// Silently ignore any errors - update check should never block or fail visibly
		checkForUpdate()
	}()
}

func checkForUpdate() {
	// Skip for dev builds
	if version == "dev" || version == "" {
		return
	}

	// Check if we should run (once per day)
	if !shouldCheckUpdate() {
		return
	}

	// Fetch latest release from GitHub
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "dbimport/"+version)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return
	}

	// Parse and compare versions
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(version, "v")

	if latestVersion != currentVersion && latestVersion != "" {
		// Parse published date
		publishedAt, _ := time.Parse(time.RFC3339, release.PublishedAt)
		publishedStr := publishedAt.Format("2006-01-02")

		// Print update warning
		fmt.Println()
		fmt.Println(warningStyle.Render("⚠️  Nouvelle version disponible !"))
		fmt.Printf("   Version : %s\n", infoStyle.Render(latestVersion))
		fmt.Printf("   Publiée : %s\n", infoStyle.Render(publishedStr))
		fmt.Printf("   URL     : %s\n", infoStyle.Render(release.HTMLURL))
		fmt.Println()
	}

	// Update last check timestamp (even if no update found)
	recordUpdateCheck(release.TagName)
}

func shouldCheckUpdate() bool {
	configDir := getConfigDir()
	data, err := os.ReadFile(filepath.Join(configDir, updateCheckFile))
	if err != nil {
		return true // Never checked before
	}

	var check UpdateCheck
	if err := json.Unmarshal(data, &check); err != nil {
		return true
	}

	return time.Since(check.LastCheck) > checkInterval
}

func recordUpdateCheck(lastVersion string) {
	check := UpdateCheck{
		LastCheck:   time.Now(),
		LastVersion: lastVersion,
	}

	data, err := json.Marshal(check)
	if err != nil {
		return
	}

	configDir := getConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return
	}

	os.WriteFile(filepath.Join(configDir, updateCheckFile), data, 0644)
}
