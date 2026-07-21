package cmd

import (
	"fmt"

	"MisakaMailClient/internal/output"

	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <email>",
	Short: "Switch the current account",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if _, ok := cfg.Find(email); !ok {
			return fmt.Errorf("account %q not found; run 'misaka-mail login' first", email)
		}
		cfg.CurrentAccount = email
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		if jsonMode {
			return output.PrintJSON(map[string]string{"current_account": email})
		}
		fmt.Printf("Switched to %s.\n", email)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
}
