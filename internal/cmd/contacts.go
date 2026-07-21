package cmd

import (
	"fmt"

	"MisakaMailClient/internal/contacts"
	"MisakaMailClient/internal/imapclient"
	"MisakaMailClient/internal/output"

	"github.com/spf13/cobra"
)

var (
	contactsLimit  int
	contactsInbox  bool
)

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Pull contacts from the mailbox (Contacts folder + sent mail)",
	Long: "Pull contacts from the mailbox. Tries a Contacts folder (vCard)\n" +
		"first, then scans the Sent folder for recipients. Use --include-inbox\n" +
		"to also collect senders from INBOX. Since IMAP has no standard address\n" +
		"book API, the Sent-derived list is the portable fallback.",
	RunE: func(cmd *cobra.Command, args []string) error {
		acc, err := resolveAccount()
		if err != nil {
			return err
		}
		pwd, err := accountPassword(acc)
		if err != nil {
			return err
		}
		conn, err := imapclient.Dial(acc.IMAP, acc.Email, pwd)
		if err != nil {
			return err
		}
		defer conn.Close()
		list, notes, err := contacts.Pull(conn, contacts.Options{
			Limit:        contactsLimit,
			IncludeInbox: contactsInbox,
		})
		if err != nil {
			return err
		}
		if jsonMode {
			return output.PrintJSON(map[string]interface{}{
				"account":  acc.Email,
				"count":    len(list),
				"contacts": list,
				"notes":    notes,
			})
		}
		fmt.Printf("Contacts for %s (%d):\n", acc.Email, len(list))
		for _, c := range list {
			if c.Name != "" {
				fmt.Printf("  %s <%s> [%s]\n", c.Name, c.Email, c.Source)
			} else {
				fmt.Printf("  %s [%s]\n", c.Email, c.Source)
			}
		}
		if len(notes) > 0 {
			fmt.Println("\nNotes:")
			for _, n := range notes {
				fmt.Println("  - " + n)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(contactsCmd)
	contactsCmd.Flags().IntVar(&contactsLimit, "limit", 200, "max messages to scan per folder (0 = all)")
	contactsCmd.Flags().BoolVar(&contactsInbox, "include-inbox", false, "also collect senders from INBOX")
}
