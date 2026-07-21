package cmd

import (
	"fmt"

	"MisakaMailClient/internal/output"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the current account",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		a, ok := cfg.Current()
		if !ok {
			return fmt.Errorf("no account selected; run 'misaka-mail login' or 'misaka-mail use <email>'")
		}
		if jsonMode {
			return output.PrintJSON(map[string]interface{}{
				"email":        a.Email,
				"provider":     a.Provider,
				"display_name": a.DisplayName,
				"imap":         a.IMAP,
				"smtp":         a.SMTP,
			})
		}
		fmt.Printf("Current account: %s\n", a.Email)
		if a.DisplayName != "" {
			fmt.Printf("  Name:     %s\n", a.DisplayName)
		}
		fmt.Printf("  Provider: %s\n", a.Provider)
		fmt.Printf("  IMAP:     %s:%d (ssl=%v)\n", a.IMAP.Host, a.IMAP.Port, a.IMAP.SSL)
		fmt.Printf("  SMTP:     %s:%d (ssl=%v)\n", a.SMTP.Host, a.SMTP.Port, a.SMTP.SSL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
