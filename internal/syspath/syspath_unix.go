//go:build !windows

package syspath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AddToUserPath appends an `export PATH=...` line for dir to the user's shell
// startup files (~/.bashrc, ~/.zshrc, ~/.profile) when they exist. It returns
// whether the path was added. Unlike Windows, Unix has no single per-user PATH
// variable, so shell rc files are edited instead.
func AddToUserPath(dir string) (bool, error) {
	dir = filepath.Clean(dir)
	home, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Errorf("determine home directory: %w", err)
	}

	candidates := []string{
		filepath.Join(home, ".bashrc"),
		filepath.Join(home, ".zshrc"),
		filepath.Join(home, ".profile"),
	}

	// If dir is already referenced in any rc file, consider it done.
	for _, rc := range candidates {
		data, err := os.ReadFile(rc)
		if err == nil && strings.Contains(string(data), dir) {
			return false, nil
		}
	}

	appended := false
	line := fmt.Sprintf("\n# Added by misaka-mail install\nexport PATH=\"$PATH:%s\"\n", dir)
	for _, rc := range candidates {
		if info, err := os.Stat(rc); err != nil || info.IsDir() {
			continue
		}
		f, err := os.OpenFile(rc, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			continue
		}
		_, err = f.WriteString(line)
		_ = f.Close()
		if err == nil {
			appended = true
		}
	}

	if !appended {
		return false, fmt.Errorf("could not edit a shell profile automatically; add this line to your shell rc:\n  export PATH=\"$PATH:%s\"", dir)
	}
	return true, nil
}
