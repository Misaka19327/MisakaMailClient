package cmd

import (
	"fmt"
	"strings"

	"MisakaMailClient/internal/imapclient"
	"MisakaMailClient/internal/message"
	"MisakaMailClient/internal/output"
	"MisakaMailClient/internal/smtpclient"

	"github.com/spf13/cobra"
)

var (
	replyAll      bool
	replyBody     string
	replyBodyFile string
	replyHTML     string
	replyHTMLFile string
	replyAttach   []string
	replyFolder   string
)

var replyCmd = &cobra.Command{
	Use:   "reply <seq>",
	Short: "Reply to a message (sets In-Reply-To and References)",
	Args:  cobra.ExactArgs(1),
	Long: "Reply to the message with the given sequence number. Threading headers\n" +
		"(In-Reply-To, References) and a 'Re:' subject are set automatically. Use\n" +
		"--all to reply to all recipients.",
	RunE: func(cmd *cobra.Command, args []string) error {
		seq, err := parseSeq(args[0])
		if err != nil {
			return err
		}
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
		folder, err := conn.ResolveFolder(replyFolder)
		if err != nil {
			return err
		}
		raw, err := conn.FetchRaw(folder, seq)
		if err != nil {
			return err
		}
		orig, err := message.Parse(raw)
		if err != nil {
			return err
		}

		// Recipients.
		var to, cc []string
		if replyAll {
			to = append(to, orig.From)
			to = append(to, orig.To...)
			to = removeAddr(to, acc.Email)
			cc = removeAddr(orig.Cc, acc.Email)
		} else {
			to = []string{orig.From}
		}

		// Subject.
		subject := orig.Subject
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(subject)), "re:") {
			subject = "Re: " + subject
		}

		// Threading references.
		refs := append([]string{}, orig.References...)
		if orig.MessageID != "" {
			refs = append(refs, orig.MessageID)
		}

		textBody, err := resolveBody(replyBody, replyBodyFile)
		if err != nil {
			return err
		}
		htmlBody, err := resolveBody(replyHTML, replyHTMLFile)
		if err != nil {
			return err
		}
		if textBody == "" && htmlBody == "" {
			return fmt.Errorf("a body is required: use --body or --html (or --body-file/--html-file)")
		}

		spec := message.SendSpec{
			From:        acc.Email,
			FromName:    acc.DisplayName,
			To:          to,
			Cc:          cc,
			Subject:     subject,
			TextBody:    textBody,
			HTMLBody:    htmlBody,
			Attachments: replyAttach,
			InReplyTo:   orig.MessageID,
			References:  refs,
		}
		rawOut, msgID, err := message.Build(spec)
		if err != nil {
			return err
		}
		recipients := spec.AllRecipients()
		if err := smtpclient.Send(acc.SMTP, acc.Email, pwd, recipients, rawOut); err != nil {
			return err
		}
		if jsonMode {
			return output.PrintJSON(map[string]interface{}{
				"sent":        true,
				"account":     acc.Email,
				"message_id":  msgID,
				"in_reply_to": orig.MessageID,
				"recipients":  recipients,
			})
		}
		fmt.Printf("Reply sent (%s) to %d recipient(s).\n", msgID, len(recipients))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(replyCmd)
	replyCmd.Flags().BoolVar(&replyAll, "all", false, "reply to all recipients")
	replyCmd.Flags().StringVar(&replyBody, "body", "", "plain-text body")
	replyCmd.Flags().StringVar(&replyBodyFile, "body-file", "", "read plain-text body from file")
	replyCmd.Flags().StringVar(&replyHTML, "html", "", "HTML body")
	replyCmd.Flags().StringVar(&replyHTMLFile, "html-file", "", "read HTML body from file")
	replyCmd.Flags().StringSliceVar(&replyAttach, "attach", nil, "attachment file path (repeatable)")
	replyCmd.Flags().StringVar(&replyFolder, "folder", "", "mailbox the message lives in (default INBOX; \"sent\" resolves to the Sent folder, or pass any folder name)")
}
