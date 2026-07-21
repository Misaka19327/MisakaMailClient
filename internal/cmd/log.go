package cmd

import (
	"fmt"
	"time"

	"MisakaMailClient/internal/config"
	"MisakaMailClient/internal/logging"
	"MisakaMailClient/internal/output"

	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Encrypted logging (key, show, level, retention, purge)",
}

var logKeyCmd = &cobra.Command{
	Use:   "key",
	Short: "Set or change the log encryption key (min 6 characters)",
	RunE: func(cmd *cobra.Command, args []string) error {
		pass, err := promptPassword("Log encryption key (min 6 chars): ")
		if err != nil {
			return err
		}
		confirm, err := promptPassword("Confirm key: ")
		if err != nil {
			return err
		}
		if pass != confirm {
			return fmt.Errorf("keys do not match")
		}
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		logging.Init(cfg.LoggingConfig())
		salt, replaced, err := logging.SetKey(pass)
		if err != nil {
			return err
		}
		if cfg.Logging == nil {
			cfg.Logging = &config.Logging{}
		}
		cfg.Logging.Salt = salt
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		if replaced {
			_ = logging.PurgeAll() // old logs are undecryptable with the new key
		}
		if jsonMode {
			return output.PrintJSON(map[string]interface{}{
				"set":              true,
				"replaced_previous": replaced,
				"purged_old_logs":  replaced,
			})
		}
		fmt.Println("Log encryption key set.")
		if replaced {
			fmt.Println("Previous key replaced; old logs removed (encrypted with the old key).")
		}
		return nil
	},
}

var (
	logShowLevel string
	logShowSince string
	logShowText  bool
)

var logShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Decrypt and print log entries (JSON by default)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		logging.Init(cfg.LoggingConfig())
		minLevel := logging.LevelDebug
		if logShowLevel != "" {
			minLevel, err = logging.ParseLevel(logShowLevel)
			if err != nil {
				return err
			}
		}
		since, err := parseSince(logShowSince)
		if err != nil {
			return err
		}
		entries, err := logging.Read(minLevel, since)
		if err != nil {
			return err
		}
		if logShowText {
			for _, e := range entries {
				fmt.Printf("%s [%s] %s | %s | %s\n",
					e.Time.Format("2006-01-02 15:04:05"), e.Level, e.Command, e.Account, e.Message)
			}
			return nil
		}
		return output.PrintJSON(entries)
	},
}

var logLevelCmd = &cobra.Command{
	Use:   "level <debug|info|warn|error>",
	Short: "Set the log level (what gets written)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		l, err := logging.ParseLevel(args[0])
		if err != nil {
			return err
		}
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if cfg.Logging == nil {
			cfg.Logging = &config.Logging{}
		}
		cfg.Logging.Level = l.String()
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		logging.Init(cfg.LoggingConfig())
		if jsonMode {
			return output.PrintJSON(map[string]string{"level": l.String()})
		}
		fmt.Printf("Log level set to %s.\n", l.String())
		return nil
	},
}

var logRetentionCmd = &cobra.Command{
	Use:   "retention <days>",
	Short: "Set how many days of logs to keep",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var days int
		if _, err := fmt.Sscanf(args[0], "%d", &days); err != nil || days <= 0 {
			return fmt.Errorf("invalid days %q", args[0])
		}
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if cfg.Logging == nil {
			cfg.Logging = &config.Logging{}
		}
		cfg.Logging.RetentionDays = days
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		logging.Init(cfg.LoggingConfig())
		if jsonMode {
			return output.PrintJSON(map[string]int{"retention_days": days})
		}
		fmt.Printf("Log retention set to %d days.\n", days)
		return nil
	},
}

var logPurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Delete all log files",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := logging.PurgeAll(); err != nil {
			return err
		}
		if jsonMode {
			return output.PrintJSON(map[string]bool{"purged": true})
		}
		fmt.Println("All log files deleted.")
		return nil
	},
}

func parseSince(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		return time.Now().Add(-d), nil
	}
	for _, layout := range []string{"2006-01-02", "2006-01-02 15:04:05", time.RFC3339} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid --since %q (use e.g. 48h or 2026-07-01)", s)
}

func init() {
	rootCmd.AddCommand(logCmd)
	logCmd.AddCommand(logKeyCmd, logShowCmd, logLevelCmd, logRetentionCmd, logPurgeCmd)
	logShowCmd.Flags().StringVar(&logShowLevel, "level", "", "only show entries at/above this level")
	logShowCmd.Flags().StringVar(&logShowSince, "since", "", "only show entries since (e.g. 48h or 2026-07-01)")
	logShowCmd.Flags().BoolVar(&logShowText, "text", false, "human-readable output (default is JSON)")
}
