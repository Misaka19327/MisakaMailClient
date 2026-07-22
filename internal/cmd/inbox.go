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
	inboxFolder string
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
		folder, err := conn.ResolveFolder(inboxFolder)
		if err != nil {
			return err
		}
		envs, err := conn.List(folder, inboxLimit, inboxUnread)
		if err != nil {
			return err
		}
		if jsonMode {
			return output.PrintJSON(map[string]interface{}{
				"account":  acc.Email,
				"folder":   folder,
				"messages": envs,
			})
		}
		if len(envs) == 0 {
			fmt.Printf("%s is empty.\n", folderLabel(folder, inboxFolder))
			return nil
		}
		fmt.Printf("%s for %s (%d messages):\n\n", folderLabel(folder, inboxFolder), acc.Email, len(envs))
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
		hint := "Read with: misaka-mail read <seq>"
		if folder != "INBOX" {
			hint += " --folder " + inboxFolder
		}
		fmt.Println("\n(U = unread, [+] = has attachments)  " + hint)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(inboxCmd)
	inboxCmd.Flags().IntVar(&inboxLimit, "limit", 20, "maximum number of messages to list (0 = all)")
	inboxCmd.Flags().BoolVar(&inboxUnread, "unread", false, "only show unread messages")
	inboxCmd.Flags().StringVar(&inboxFolder, "folder", "", "mailbox to list (default INBOX; \"sent\" resolves to the Sent folder, or pass any folder name)")
}

// folderLabel returns a human-readable label for the resolved folder. INBOX
// becomes "Inbox"; any other folder uses the user-supplied input (e.g. "sent"
// or "已发送") since the resolved raw name may be UTF-7 encoded.
func folderLabel(folder, input string) string {
	if folder == "INBOX" {
		return "Inbox"
	}
	if input != "" {
		return input
	}
	return folder
}
