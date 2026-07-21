// Package output renders command results as JSON or human-readable text.
package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON writes v as indented JSON to stdout. HTML escaping is disabled so
// bodies and subjects containing '<', '>', '&' render verbatim.
func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// PrintError writes err to stderr, as a JSON object when jsonMode is true.
func PrintError(jsonMode bool, err error) {
	if err == nil {
		return
	}
	if jsonMode {
		enc := json.NewEncoder(os.Stderr)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(map[string]string{"error": err.Error()})
	} else {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
}

// Truncate shortens s to at most n runes, appending "…" when truncated.
func Truncate(s string, n int) string {
	if n <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
