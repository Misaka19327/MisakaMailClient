package skill

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const skillContent = "# misaka-mail skill\nfont-size: 13px\n"

// newServer returns an httptest server that serves skillContent as a GitHub
// Contents-API JSON response (base64-encoded) for any path, plus its base URL.
func newServer(t *testing.T, status int) (*httptest.Server, string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			http.Error(w, "not found", status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"content":  base64.StdEncoding.EncodeToString([]byte(skillContent)),
			"encoding": "base64",
			"size":     len(skillContent),
		})
	}))
	t.Cleanup(srv.Close)
	return srv, srv.URL
}

// withAPIBase temporarily points the skill package at base for the test.
func withAPIBase(t *testing.T, base string) {
	t.Helper()
	prev := apiBase
	apiBase = base
	t.Cleanup(func() { apiBase = prev })
}

func TestFetch_Success(t *testing.T) {
	_, base := newServer(t, http.StatusOK)
	withAPIBase(t, base)
	data, err := fetch(context.Background(), "owner", "repo", "v0.5.0", sourcePath)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if string(data) != skillContent {
		t.Errorf("fetch body = %q, want %q", data, skillContent)
	}
}

func TestFetch_404(t *testing.T) {
	_, base := newServer(t, http.StatusNotFound)
	withAPIBase(t, base)
	if _, err := fetch(context.Background(), "owner", "repo", "v0.5.0", sourcePath); err == nil {
		t.Fatal("fetch: expected error for 404, got nil")
	}
}

func TestWriteTo(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "SKILL.md")
	r := writeTo(dir, []byte(skillContent), "v0.5.0", false)
	if !r.Updated {
		t.Fatalf("writeTo: not updated: %s", r.Error)
	}
	if r.Path != target {
		t.Errorf("path = %q, want %q", r.Path, target)
	}
	if r.Fallback {
		t.Error("writeTo: Fallback should be false for primary dir")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != skillContent {
		t.Errorf("file content = %q, want %q", got, skillContent)
	}
}

// TestInstall_PersonalDir drives the full Install against an httptest server,
// redirecting HOME/USERPROFILE so the skill lands in a temp dir.
func TestInstall_PersonalDir(t *testing.T) {
	_, base := newServer(t, http.StatusOK)
	withAPIBase(t, base)
	home := t.TempDir()
	// os.UserHomeDir reads USERPROFILE on Windows, HOME on Unix.
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOME", home)

	r := Install(context.Background(), "owner", "repo", "v0.5.0")
	if !r.Updated {
		t.Fatalf("Install: not updated: %s", r.Error)
	}
	want := filepath.Join(home, ".claude", "skills", "misaka-mail", "SKILL.md")
	if r.Path != want {
		t.Errorf("path = %q, want %q", r.Path, want)
	}
	if r.Fallback {
		t.Error("Install: should write to the personal dir, not the fallback")
	}
	got, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != skillContent {
		t.Errorf("content = %q, want %q", got, skillContent)
	}
}

// TestInstall_FetchFailure reports an error (and does not panic) when the
// skill cannot be downloaded - the caller surfaces a notice.
func TestInstall_FetchFailure(t *testing.T) {
	_, base := newServer(t, http.StatusNotFound)
	withAPIBase(t, base)
	r := Install(context.Background(), "owner", "repo", "v0.5.0")
	if r.Updated {
		t.Fatal("Install: should not be updated on fetch failure")
	}
	if r.Error == "" {
		t.Error("Install: expected an error message on fetch failure")
	}
	if !strings.Contains(r.Error, "404") {
		t.Errorf("error %q should mention 404", r.Error)
	}
}

// TestInstall_EmptyRef guards the no-tag case.
func TestInstall_EmptyRef(t *testing.T) {
	r := Install(context.Background(), "owner", "repo", "")
	if r.Updated {
		t.Fatal("Install: should not update with empty ref")
	}
	if r.Error == "" {
		t.Error("Install: expected an error for empty ref")
	}
}

func TestManualURL(t *testing.T) {
	got := ManualURL("Misaka19327", "MisakaMailClient", "v0.5.0")
	want := "https://github.com/Misaka19327/MisakaMailClient/blob/v0.5.0/.claude/skills/misaka-mail/SKILL.md"
	if got != want {
		t.Errorf("ManualURL = %q, want %q", got, want)
	}
}
