package cmd

import (
	"fmt"

	"MisakaMailClient/internal/output"

	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "List configured accounts",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		type acctOut struct {
			Email    string `json:"email"`
			Provider string `json:"provider"`
			Name     string `json:"name,omitempty"`
			Current  bool   `json:"current"`
		}
		out := make([]acctOut, 0, len(cfg.Accounts))
		for _, a := range cfg.Accounts {
			out = append(out, acctOut{
				Email:    a.Email,
				Provider: a.Provider,
				Name:     a.DisplayName,
				Current:  a.Email == cfg.CurrentAccount || (cfg.CurrentAccount == "" && len(cfg.Accounts) == 1),
			})
		}
		if jsonMode {
			return output.PrintJSON(out)
		}
		if len(out) == 0 {
			fmt.Println("No accounts configured. Run 'misaka-mail login' to add one.")
			return nil
		}
		for _, a := range out {
			mark := "  "
			if a.Current {
				mark = "* "
			}
			if a.Name != "" {
				fmt.Printf("%s%s (%s) [%s]\n", mark, a.Email, a.Name, a.Provider)
			} else {
				fmt.Printf("%s%s [%s]\n", mark, a.Email, a.Provider)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(accountsCmd)
}
