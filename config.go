package main

import (
	"os"
	"path/filepath"
)

func getConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config", "dbimport")
}

func loadLastDir() string {
	configDir := getConfigDir()
	data, err := os.ReadFile(filepath.Join(configDir, "lastdir"))
	if err != nil {
		return "."
	}
	lastDir := string(data)
	if _, err := os.Stat(lastDir); os.IsNotExist(err) {
		return "."
	}
	return lastDir
}

func saveLastDir(dir string) error {
	configDir := getConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "lastdir"), []byte(dir), 0644)
}
