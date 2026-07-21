package cmd

import (
	"fmt"

	"MisakaMailClient/internal/credentials"
	"MisakaMailClient/internal/output"

	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout <email>",
	Short: "Remove an account and delete its stored password",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if !cfg.Remove(email) {
			return fmt.Errorf("account %q not found", email)
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		// Best-effort: remove the password from the keyring too.
		_ = credentials.DeletePassword(email)
		if jsonMode {
			return output.PrintJSON(map[string]string{"removed": email})
		}
		fmt.Printf("Removed account %s and its stored password.\n", email)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
