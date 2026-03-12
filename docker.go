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

type ContainerCredentials struct {
	DBName     string
	DBUser     string
	DBPassword string
	Source     string // "env" or "saved" to track where it came from
}

func extractContainerEnvVars(container string) (map[string]string, error) {
	out, err := exec.Command(
		"docker", "inspect",
		"--format", "{{range .Config.Env}}{{.}}\n{{end}}",
		container,
	).Output()
	if err != nil {
		return nil, err
	}

	envVars := make(map[string]string)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}
	return envVars, nil
}

func getCredentialsFromContainer(container string, dbType string) *ContainerCredentials {
	envVars, err := extractContainerEnvVars(container)
	if err != nil {
		return nil
	}

	creds := &ContainerCredentials{Source: "env"}

	switch dbType {
	case "pgsql":
		// PostgreSQL official image variables
		if val, ok := envVars["POSTGRES_DB"]; ok {
			creds.DBName = val
		}
		if val, ok := envVars["POSTGRES_USER"]; ok {
			creds.DBUser = val
		}
		if val, ok := envVars["POSTGRES_PASSWORD"]; ok {
			creds.DBPassword = val
		}
		// Fallback to common alternatives
		if creds.DBName == "" {
			if val, ok := envVars["PGDATABASE"]; ok {
				creds.DBName = val
			}
		}
		if creds.DBUser == "" {
			if val, ok := envVars["PGUSER"]; ok {
				creds.DBUser = val
			}
		}
		if creds.DBPassword == "" {
			if val, ok := envVars["PGPASSWORD"]; ok {
				creds.DBPassword = val
			}
		}
	case "mysql":
		// MySQL/MariaDB official image variables
		if val, ok := envVars["MYSQL_DATABASE"]; ok {
			creds.DBName = val
		} else if val, ok := envVars["MARIADB_DATABASE"]; ok {
			creds.DBName = val
		}
		if val, ok := envVars["MYSQL_USER"]; ok {
			creds.DBUser = val
		} else if val, ok := envVars["MARIADB_USER"]; ok {
			creds.DBUser = val
		}
		if val, ok := envVars["MYSQL_PASSWORD"]; ok {
			creds.DBPassword = val
		} else if val, ok := envVars["MARIADB_PASSWORD"]; ok {
			creds.DBPassword = val
		}
		// Fallback to root password if user password not set
		if creds.DBPassword == "" {
			if val, ok := envVars["MYSQL_ROOT_PASSWORD"]; ok {
				creds.DBUser = "root"
				creds.DBPassword = val
			} else if val, ok := envVars["MARIADB_ROOT_PASSWORD"]; ok {
				creds.DBUser = "root"
				creds.DBPassword = val
			}
		}
	}

	// Only return if we got at least user or password
	if creds.DBUser != "" || creds.DBPassword != "" {
		return creds
	}
	return nil
}
