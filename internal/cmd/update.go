package cmd

import (
	"context"
	"fmt"
	"strings"

	"MisakaMailClient/internal/output"
	"MisakaMailClient/internal/skill"
	"MisakaMailClient/internal/updater"

	"github.com/spf13/cobra"
)

var (
	updateCheck    bool
	updateVersion  string
	updateRepo     string
	updateNoVerify bool
	updateNoSkill  bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update misaka-mail to the latest GitHub release",
	Long: "Update misaka-mail to the latest release from GitHub. The running\n" +
		"binary is replaced atomically. Downloads are SHA256-verified against the\n" +
		"release's checksums.txt unless --no-verify is set.\n\n" +
		"After the binary is updated, the Claude skill (SKILL.md) for the same\n" +
		"release is fetched and written to ~/.claude/skills/misaka-mail/SKILL.md\n" +
		"so Claude Code loads the matching skill. Use --no-skill to skip this.\n\n" +
		"The GitHub repository is set at build time (via -ldflags) or overridden\n" +
		"with --repo owner/repo. Set GITHUB_TOKEN for higher API rate limits or\n" +
		"private repositories.",
	RunE: func(cmd *cobra.Command, args []string) error {
		owner, repo := updater.DefaultRepo()
		if updateRepo != "" {
			o, r, err := parseRepo(updateRepo)
			if err != nil {
				return err
			}
			owner, repo = o, r
		}
		if owner == "" || repo == "" {
			return fmt.Errorf("update repository not configured; pass --repo owner/repo, or rebuild with -ldflags setting the GitHub owner/repo (see README)")
		}
		ctx := context.Background()
		verify := !updateNoVerify

		// Specific version path.
		if updateVersion != "" {
			rel, found, err := updater.CheckVersion(ctx, owner, repo, updateVersion, verify)
			if err != nil {
				return fmt.Errorf("detect version: %w", err)
			}
			if !found {
				return fmt.Errorf("version %s not found in %s/%s", updateVersion, owner, repo)
			}
			if updateCheck {
				return emitReleaseCheck(updater.ToReleaseInfo(rel), true)
			}
			if err := updater.Apply(ctx, rel, verify); err != nil {
				return fmt.Errorf("update to %s: %w", updateVersion, err)
			}
			return finishUpdate(ctx, updater.ToReleaseInfo(rel), owner, repo)
		}

		// Latest path.
		rel, newer, found, err := updater.CheckLatest(ctx, owner, repo, appVersion, verify)
		if err != nil {
			return fmt.Errorf("check latest release: %w", err)
		}
		if !found || rel == nil {
			return fmt.Errorf("no releases found in %s/%s", owner, repo)
		}
		if updateCheck {
			return emitReleaseCheck(updater.ToReleaseInfo(rel), newer)
		}
		if !newer {
			if jsonMode {
				return output.PrintJSON(map[string]interface{}{
					"current":          appVersion,
					"latest":           rel.Version(),
					"updated":          false,
					"update_available": false,
				})
			}
			fmt.Printf("Already up to date (current %s, latest %s).\n", appVersion, rel.Version())
			return nil
		}
		if err := updater.Apply(ctx, rel, verify); err != nil {
			return fmt.Errorf("update: %w", err)
		}
		return finishUpdate(ctx, updater.ToReleaseInfo(rel), owner, repo)
	},
}

func parseRepo(s string) (string, string, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid --repo %q, expected owner/repo", s)
	}
	return parts[0], parts[1], nil
}

func emitReleaseCheck(info updater.ReleaseInfo, newer bool) error {
	if jsonMode {
		return output.PrintJSON(map[string]interface{}{
			"current":          appVersion,
			"latest":           info.Version,
			"update_available": newer,
			"release":          info,
		})
	}
	if newer {
		fmt.Printf("Update available: %s -> %s\n", appVersion, info.Version)
		if info.URL != "" {
			fmt.Printf("  %s\n", info.URL)
		}
	} else {
		fmt.Printf("No update available (current %s, latest %s).\n", appVersion, info.Version)
	}
	return nil
}

// finishUpdate runs the post-update skill sync (unless --no-skill) and emits
// the update result, including the skill outcome.
func finishUpdate(ctx context.Context, info updater.ReleaseInfo, owner, repo string) error {
	var skillRes skill.Result
	if !updateNoSkill {
		skillRes = skill.Install(ctx, owner, repo, releaseTag(info))
	}
	return emitUpdated(info, skillRes, !updateNoSkill, owner, repo)
}

// releaseTag extracts the git tag of the release from its URL (after "/tag/"),
// falling back to "v"+version. Used as the ref to fetch the matching skill.
func releaseTag(info updater.ReleaseInfo) string {
	if i := strings.Index(info.URL, "/tag/"); i >= 0 {
		return strings.TrimSuffix(info.URL[i+len("/tag/"):], "/")
	}
	if info.Version == "" {
		return ""
	}
	if strings.HasPrefix(info.Version, "v") {
		return info.Version
	}
	return "v" + info.Version
}

// skillNotice returns the human-readable line(s) describing the skill sync
// outcome, printed after the binary-update line.
func skillNotice(r skill.Result, owner, repo, ref string) string {
	switch {
	case r.Updated && !r.Fallback:
		return fmt.Sprintf("Skill updated: %s\n", r.Path)
	case r.Updated && r.Fallback:
		return fmt.Sprintf("Skill saved to %s (could not write to ~/.claude/skills).\n", r.Path) +
			fmt.Sprintf("Copy it to ~/.claude/skills/misaka-mail/SKILL.md to enable in Claude Code.\n")
	default:
		return fmt.Sprintf("Skill: could not update automatically (%s).\n", r.Error) +
			fmt.Sprintf("Download manually: %s\n", skill.ManualURL(owner, repo, ref))
	}
}

func emitUpdated(info updater.ReleaseInfo, skillRes skill.Result, skillEnabled bool, owner, repo string) error {
	if jsonMode {
		m := map[string]interface{}{
			"updated": true,
			"version": info.Version,
		}
		if skillEnabled {
			m["skill"] = skillRes
		}
		return output.PrintJSON(m)
	}
	fmt.Printf("Updated to %s. Restart misaka-mail to use the new version.\n", info.Version)
	if skillEnabled {
		fmt.Print(skillNotice(skillRes, owner, repo, releaseTag(info)))
	}
	return nil
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "only check whether an update is available; do not apply it")
	updateCmd.Flags().StringVar(&updateVersion, "version", "", "update to a specific version (e.g. 0.5.0 or v0.5.0)")
	updateCmd.Flags().StringVar(&updateRepo, "repo", "", "GitHub repository as owner/repo (overrides the build-time default)")
	updateCmd.Flags().BoolVar(&updateNoVerify, "no-verify", false, "skip SHA256 checksum verification (insecure)")
	updateCmd.Flags().BoolVar(&updateNoSkill, "no-skill", false, "do not fetch the Claude skill after updating the binary")
}
