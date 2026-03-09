package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

type ContainerInfo struct {
	Name   string
	Image  string
	Status string
	Ports  string
}

func isDBContainer(image string) bool {
	img := strings.ToLower(image)
	return strings.Contains(img, "mysql") ||
		strings.Contains(img, "mariadb") ||
		strings.Contains(img, "postgres") ||
		strings.Contains(img, "mongo") ||
		strings.Contains(img, "redis") ||
		strings.Contains(img, "elasticsearch") ||
		strings.Contains(img, "cassandra") ||
		strings.Contains(img, "couchdb")
}

func listContainers() ([]ContainerInfo, error) {
	out, err := exec.Command(
		"docker", "ps",
		"--format", "{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}",
	).Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []ContainerInfo
	for _, l := range lines {
		if l = strings.TrimSpace(l); l != "" {
			parts := strings.Split(l, "\t")
			if len(parts) >= 1 {
				info := ContainerInfo{Name: parts[0]}
				if len(parts) >= 2 {
					info.Image = parts[1]
				}
				if len(parts) >= 3 {
					info.Status = parts[2]
				}
				if len(parts) >= 4 {
					info.Ports = parts[3]
				}
				// Filter only DB containers
				if isDBContainer(info.Image) {
					result = append(result, info)
				}
			}
		}
	}
	return result, nil
}

func containerToOptions(containers []ContainerInfo) []huh.Option[string] {
	grayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	opts := make([]huh.Option[string], len(containers))
	for i, c := range containers {
		label := c.Name
		info := fmt.Sprintf("  %s | %s", c.Image, c.Status)
		if c.Ports != "" {
			info += fmt.Sprintf(" | %s", c.Ports)
		}
		display := label + grayStyle.Render(info)
		opts[i] = huh.NewOption(display, c.Name)
	}
	return opts
}

func detectDBType(container string) string {
	out, err := exec.Command(
		"docker", "inspect",
		"--format", "{{.Config.Image}}",
		container,
	).Output()
	if err != nil {
		return ""
	}
	image := strings.ToLower(string(out))
	if strings.Contains(image, "mysql") || strings.Contains(image, "mariadb") {
		return "mysql"
	}
	if strings.Contains(image, "postgres") {
		return "pgsql"
	}
	return ""
}
