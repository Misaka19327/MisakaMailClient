package cmd

import (
	"fmt"

	"MisakaMailClient/internal/message"
	"MisakaMailClient/internal/output"
	"MisakaMailClient/internal/smtpclient"

	"github.com/spf13/cobra"
)

var (
	sendTo        []string
	sendCc        []string
	sendBcc       []string
	sendAttach    []string
	sendSubject   string
	sendBody      string
	sendBodyFile  string
	sendHTML      string
	sendHTMLFile  string
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send a new email",
	Long: "Send a new email. --to is repeatable (or comma-separated); supply a body via\n" +
		"--body/--html (inline) or --body-file/--html-file (from a file); attach files\n" +
		"with --attach (repeatable).",
	RunE: func(cmd *cobra.Command, args []string) error {
		acc, err := resolveAccount()
		if err != nil {
			return err
		}
		pwd, err := accountPassword(acc)
		if err != nil {
			return err
		}
		textBody, err := resolveBody(sendBody, sendBodyFile)
		if err != nil {
			return err
		}
		htmlBody, err := resolveBody(sendHTML, sendHTMLFile)
		if err != nil {
			return err
		}
		if textBody == "" && htmlBody == "" {
			return fmt.Errorf("a body is required: use --body or --html (or --body-file/--html-file)")
		}
		if len(sendTo)+len(sendCc)+len(sendBcc) == 0 {
			return fmt.Errorf("at least one recipient is required: use --to, --cc, or --bcc")
		}
		spec := message.SendSpec{
			From:        acc.Email,
			FromName:    acc.DisplayName,
			To:          sendTo,
			Cc:          sendCc,
			Bcc:         sendBcc,
			Subject:     sendSubject,
			TextBody:    textBody,
			HTMLBody:    htmlBody,
			Attachments: sendAttach,
		}
		raw, msgID, err := message.Build(spec)
		if err != nil {
			return err
		}
		recipients := spec.AllRecipients()
		if err := smtpclient.Send(acc.SMTP, acc.Email, pwd, recipients, raw); err != nil {
			return err
		}
		if jsonMode {
			return output.PrintJSON(map[string]interface{}{
				"sent":       true,
				"account":    acc.Email,
				"message_id": msgID,
				"recipients": recipients,
			})
		}
		fmt.Printf("Sent message %s from %s to %d recipient(s).\n", msgID, acc.Email, len(recipients))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(sendCmd)
	sendCmd.Flags().StringSliceVar(&sendTo, "to", nil, "recipient (repeatable, or comma-separated)")
	sendCmd.Flags().StringSliceVar(&sendCc, "cc", nil, "cc recipient (repeatable)")
	sendCmd.Flags().StringSliceVar(&sendBcc, "bcc", nil, "bcc recipient (repeatable)")
	sendCmd.Flags().StringVar(&sendSubject, "subject", "", "subject")
	sendCmd.Flags().StringVar(&sendBody, "body", "", "plain-text body")
	sendCmd.Flags().StringVar(&sendBodyFile, "body-file", "", "read plain-text body from file")
	sendCmd.Flags().StringVar(&sendHTML, "html", "", "HTML body")
	sendCmd.Flags().StringVar(&sendHTMLFile, "html-file", "", "read HTML body from file")
	sendCmd.Flags().StringSliceVar(&sendAttach, "attach", nil, "attachment file path (repeatable)")
}
