package main

import (
	"os"
	"strings"
)

func listFiles(dir string, exts []string) ([]string, error) {
	var result []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		for _, ext := range exts {
			if strings.HasSuffix(name, ext) {
				result = append(result, name)
				break
			}
		}
	}
	return result, nil
}
