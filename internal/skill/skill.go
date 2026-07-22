// Package skill syncs the misaka-mail Claude skill (SKILL.md) from a GitHub
// release to the user's personal Claude skills directory. It is invoked after
// a self-update so that refreshing the binary also refreshes the skill Claude
// Code loads.
//
// The skill is fetched at the installed release's tag (not main) so the skill
// stays version-aligned with the binary. Fetch and write failures never abort
// the update - the binary has already been replaced - they are reported via
// Result so the caller can surface a notice.
package skill

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// skillName is the skill directory name, both in the repo and under
	// ~/.claude/skills/.
	skillName = "misaka-mail"
	// sourcePath is the path to the skill file inside the repository.
	sourcePath = ".claude/skills/misaka-mail/SKILL.md"
)

// apiBase is the GitHub API host used to fetch the skill. The Contents API is
// used (rather than raw.githubusercontent.com) because it is reachable in
// environments where the raw CDN host is blocked. It is a var so tests can
// redirect it at an httptest server.
var apiBase = "https://api.github.com"

// Result describes the outcome of a skill sync.
type Result struct {
	Updated  bool   `json:"updated"`            // the skill file was written
	Path     string `json:"path,omitempty"`     // where it was written
	Fallback bool   `json:"fallback,omitempty"` // path is the app-data fallback, not ~/.claude
	Version  string `json:"version,omitempty"`  // git ref (release tag) used
	Error    string `json:"error,omitempty"`    // fetch or write failed
	Skipped  bool   `json:"skipped,omitempty"`  // sync was skipped (e.g. --no-skill)
}

// Install fetches the SKILL.md at ref (a release tag) from owner/repo and
// writes it to the user's personal Claude skills directory
// (~/.claude/skills/misaka-mail/SKILL.md), falling back to the app data dir
// (<userconfig>/misaka-mail/skills/misaka-mail/SKILL.md) if that is not
// writable. It never returns an error that should fail the overall update;
// failures are reported via Result.Error for display.
func Install(ctx context.Context, owner, repo, ref string) Result {
	if ref == "" {
		return Result{Error: "no release tag to fetch skill from"}
	}
	data, err := fetch(ctx, owner, repo, ref, sourcePath)
	if err != nil {
		return Result{Version: ref, Error: err.Error()}
	}
	// Primary: personal Claude skills dir (~/.claude/skills/misaka-mail).
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if r := writeTo(filepath.Join(home, ".claude", "skills", skillName), data, ref, false); r.Updated {
			return r
		}
	}
	// Fallback: app data dir. Not auto-loaded by Claude Code, but saved
	// somewhere the user can find and wire up manually.
	if cfg, err := os.UserConfigDir(); err == nil && cfg != "" {
		return writeTo(filepath.Join(cfg, "misaka-mail", "skills", skillName), data, ref, true)
	}
	return Result{Version: ref, Error: "no writable skill location found"}
}

// contentsResp is the subset of the GitHub Contents API response that fetch
// uses. For files up to 1 MB the content is returned inline as base64.
type contentsResp struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

// fetch downloads path at ref from the GitHub Contents API
// ({apiBase}/repos/{owner}/{repo}/contents/{path}?ref={ref}) and base64-decodes
// it. The Contents API is used instead of raw.githubusercontent.com because it
// is reachable in environments where the raw CDN host is blocked. A
// User-Agent header is required by the GitHub API. If GITHUB_TOKEN is set it is
// sent as a bearer token (for private repos and higher rate limits).
func fetch(ctx context.Context, owner, repo, ref, path string) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", apiBase, owner, repo, path, ref)
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "misaka-mail")
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch skill: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch skill: %s: %s", url, resp.Status)
	}
	var cr contentsResp
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("decode skill response: %w", err)
	}
	if cr.Encoding != "base64" || cr.Content == "" {
		return nil, fmt.Errorf("fetch skill: unexpected encoding %q (file may be too large; download manually)", cr.Encoding)
	}
	data, err := base64.StdEncoding.DecodeString(cr.Content)
	if err != nil {
		return nil, fmt.Errorf("decode skill base64: %w", err)
	}
	return data, nil
}

// writeTo writes data as SKILL.md inside dir, creating dir if needed. fallback
// marks whether dir is the app-data fallback location (rather than ~/.claude).
func writeTo(dir string, data []byte, ref string, fallback bool) Result {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{Version: ref, Fallback: fallback, Error: "create " + dir + ": " + err.Error()}
	}
	path := filepath.Join(dir, "SKILL.md")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return Result{Version: ref, Fallback: fallback, Error: "write " + path + ": " + err.Error()}
	}
	return Result{Updated: true, Path: path, Version: ref, Fallback: fallback}
}

// ManualURL returns a human-friendly URL for downloading the skill by hand at
// ref, used in notices when the automatic sync fails.
func ManualURL(owner, repo, ref string) string {
	return fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", owner, repo, ref, sourcePath)
}
