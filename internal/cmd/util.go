package cmd

import (
	"fmt"
	"os"
	"strings"

	"MisakaMailClient/internal/config"
)

// loadConfig reads the configuration with a wrapped error.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return cfg, nil
}

// resolveBody returns inline if non-empty, otherwise reads the body from file.
func resolveBody(inline, file string) (string, error) {
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", file, err)
		}
		return string(data), nil
	}
	return inline, nil
}

// removeAddr returns addrs with any address equal to skip (case-insensitive)
// removed; empty entries are dropped.
func removeAddr(addrs []string, skip string) []string {
	out := make([]string, 0, len(addrs))
	skip = strings.TrimSpace(skip)
	for _, a := range addrs {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if skip != "" && strings.EqualFold(a, skip) {
			continue
		}
		out = append(out, a)
	}
	return out
}
