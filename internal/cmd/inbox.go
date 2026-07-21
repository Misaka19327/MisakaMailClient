package cmd

import (
	"fmt"

	"MisakaMailClient/internal/imapclient"
	"MisakaMailClient/internal/output"

	"github.com/spf13/cobra"
)

var (
	inboxLimit  int
	inboxUnread bool
)

var inboxCmd = &cobra.Command{
	Use:   "inbox",
	Short: "List messages in the inbox",
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
		envs, err := conn.Inbox(inboxLimit, inboxUnread)
		if err != nil {
			return err
		}
		if jsonMode {
			return output.PrintJSON(map[string]interface{}{
				"account":  acc.Email,
				"messages": envs,
			})
		}
		if len(envs) == 0 {
			fmt.Println("Inbox is empty.")
			return nil
		}
		fmt.Printf("Inbox for %s (%d messages):\n\n", acc.Email, len(envs))
		for _, e := range envs {
			mark := " "
			if !e.Seen {
				mark = "U"
			}
			from := e.FromName
			if from == "" {
				from = e.From
			}
			subj := output.Truncate(e.Subject, 50)
			if e.HasAttachments {
				subj += " [+]"
			}
			fmt.Printf("%s %4d  %-25s  %-24s  %s\n", mark, e.Seq, output.Truncate(from, 25), output.Truncate(e.Date, 24), subj)
		}
		fmt.Println("\n(U = unread, [+] = has attachments)  Read with: misaka-mail read <seq>")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(inboxCmd)
	inboxCmd.Flags().IntVar(&inboxLimit, "limit", 20, "maximum number of messages to list (0 = all)")
	inboxCmd.Flags().BoolVar(&inboxUnread, "unread", false, "only show unread messages")
}
