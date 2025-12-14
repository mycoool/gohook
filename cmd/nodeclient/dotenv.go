package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// loadDotEnvFiles loads variables from .env style files without overriding already-set env vars.
// Priority (later wins only if env var is still unset):
// 1) explicit envFile
// 2) <dataDir>/.env
// 3) ./.env
func loadDotEnvFiles(envFile string, dataDir string) error {
	paths := make([]string, 0, 3)
	if strings.TrimSpace(envFile) != "" {
		paths = append(paths, envFile)
	}
	if strings.TrimSpace(dataDir) != "" {
		paths = append(paths, filepath.Join(dataDir, ".env"))
	}
	paths = append(paths, ".env")

	var lastErr error
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := loadDotEnvFile(p); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func loadDotEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)

		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
	}
	return scanner.Err()
}
