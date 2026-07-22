// Package updater implements GitHub-Releases-based self-update.
//
// The running binary is replaced atomically with the latest release asset from
// a GitHub repository. Downloads are verified against a checksums.txt asset
// (produced by the go-selfupdate publishing tool) when verification is enabled.
package updater

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
)

// defaultOwner and defaultRepo are the GitHub repository to check for releases.
// Override at build time:
//
//	go install -ldflags "-X MisakaMailClient/internal/updater.defaultOwner=OWNER -X MisakaMailClient/internal/updater.defaultRepo=REPO" ./cmd/misaka-mail
var (
	defaultOwner = "Misaka19327"
	defaultRepo  = "MisakaMailClient"
)

// DefaultRepo returns the configured GitHub owner and repo. Either may be empty
// if not set at build time.
func DefaultRepo() (owner, repo string) {
	return defaultOwner, defaultRepo
}

// ReleaseInfo is a summary of an available release.
type ReleaseInfo struct {
	Version     string `json:"version"`
	URL         string `json:"url,omitempty"`
	Name        string `json:"name,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	Prerelease  bool   `json:"prerelease,omitempty"`
	Notes       string `json:"notes,omitempty"`
}

func ToReleaseInfo(r *selfupdate.Release) ReleaseInfo {
	if r == nil {
		return ReleaseInfo{}
	}
	info := ReleaseInfo{
		Version:    r.Version(),
		URL:        r.URL,
		Name:       r.Name,
		Prerelease: r.Prerelease,
		Notes:      r.ReleaseNotes,
	}
	if !r.PublishedAt.IsZero() {
		info.PublishedAt = r.PublishedAt.Format("2006-01-02 15:04:05")
	}
	return info
}

func newUpdater(verify bool) (*selfupdate.Updater, error) {
	cfg := selfupdate.Config{}
	if verify {
		// Verify the downloaded binary against a checksums.txt asset. The
		// go-selfupdate publishing tool generates this file by default.
		cfg.Validator = &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"}
	}
	return selfupdate.NewUpdater(cfg)
}

// isNewer reports whether rel is newer than current. A non-semver current
// (e.g. "dev") is treated as outdated so an update is offered.
func isNewer(rel *selfupdate.Release, current string) bool {
	if rel == nil {
		return false
	}
	if _, err := semver.NewVersion(current); err != nil {
		return true
	}
	return rel.GreaterThan(current)
}

// CheckLatest returns the latest release and whether it is newer than current.
// found is false when the repository has no releases.
func CheckLatest(ctx context.Context, owner, repo, current string, verify bool) (rel *selfupdate.Release, newer bool, found bool, err error) {
	up, err := newUpdater(verify)
	if err != nil {
		return nil, false, false, err
	}
	rel, found, err = up.DetectLatest(ctx, selfupdate.NewRepositorySlug(owner, repo))
	if err != nil || !found {
		return rel, false, found, err
	}
	return rel, isNewer(rel, current), true, nil
}

// CheckVersion returns the release for a specific version. go-selfupdate
// matches the version against the release tag name with exact string equality,
// so both "0.5.0" and "v0.5.0" are accepted: if the input does not match, the
// other "v"-prefix variant is tried.
func CheckVersion(ctx context.Context, owner, repo, version string, verify bool) (rel *selfupdate.Release, found bool, err error) {
	up, err := newUpdater(verify)
	if err != nil {
		return nil, false, err
	}
	repoSlug := selfupdate.NewRepositorySlug(owner, repo)
	rel, found, err = up.DetectVersion(ctx, repoSlug, version)
	if err != nil || found {
		return rel, found, err
	}
	if alt := toggleVPrefix(version); alt != "" && alt != version {
		return up.DetectVersion(ctx, repoSlug, alt)
	}
	return rel, found, err
}

// toggleVPrefix returns version with a leading "v" added if absent, or removed
// if present, so a version can be matched against either tag convention. It
// returns the empty string for empty input.
func toggleVPrefix(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}
	if strings.HasPrefix(version, "v") || strings.HasPrefix(version, "V") {
		return version[1:]
	}
	return "v" + version
}

// Apply downloads rel and atomically replaces the running binary with it.
func Apply(ctx context.Context, rel *selfupdate.Release, verify bool) error {
	if rel == nil {
		return fmt.Errorf("no release to apply")
	}
	up, err := newUpdater(verify)
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate current executable: %w", err)
	}
	return up.UpdateTo(ctx, rel, exe)
}
