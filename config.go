package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ContainerConfig struct {
	DBName     string `json:"dbName"`
	DBUser     string `json:"dbUser"`
	DBPassword string `json:"dbPassword"`
}

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

func loadContainerConfig(containerName string) *ContainerConfig {
	configDir := getConfigDir()
	data, err := os.ReadFile(filepath.Join(configDir, "containers", containerName+".json"))
	if err != nil {
		return nil
	}
	
	var cfg ContainerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

func saveContainerConfig(containerName string, cfg *ContainerConfig) error {
	configDir := getConfigDir()
	containersDir := filepath.Join(configDir, "containers")
	if err := os.MkdirAll(containersDir, 0755); err != nil {
		return err
	}
	
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	
	return os.WriteFile(filepath.Join(containersDir, containerName+".json"), data, 0600)
}
