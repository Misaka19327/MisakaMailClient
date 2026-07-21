package cmd

import (
	"fmt"
	"strings"

	"MisakaMailClient/internal/imapclient"
	"MisakaMailClient/internal/message"
	"MisakaMailClient/internal/output"

	"github.com/spf13/cobra"
)

var readSaveAttachments string

var readCmd = &cobra.Command{
	Use:   "read <seq>",
	Short: "Read a message (headers, body, and attachments)",
	Args:  cobra.ExactArgs(1),
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
		raw, err := conn.FetchRaw(seq)
		if err != nil {
			return err
		}
		pm, err := message.Parse(raw)
		if err != nil {
			return err
		}
		var saved []message.SavedAttachment
		if readSaveAttachments != "" {
			saved, err = message.SaveAttachments(raw, readSaveAttachments)
			if err != nil {
				return fmt.Errorf("save attachments: %w", err)
			}
		}
		if jsonMode {
			result := map[string]interface{}{"account": acc.Email, "message": pm}
			if saved != nil {
				result["saved_attachments"] = saved
			}
			return output.PrintJSON(result)
		}
		printMessageText(pm)
		if len(saved) > 0 {
			fmt.Println("\nSaved attachments:")
			for _, s := range saved {
				fmt.Printf("  - %s -> %s (%d bytes)\n", s.Filename, s.Path, s.Size)
			}
		}
		return nil
	},
}

func printMessageText(pm *message.ParsedMessage) {
	fmt.Printf("Subject:    %s\n", pm.Subject)
	if pm.FromName != "" {
		fmt.Printf("From:       %s <%s>\n", pm.FromName, pm.From)
	} else {
		fmt.Printf("From:       %s\n", pm.From)
	}
	fmt.Printf("To:         %s\n", strings.Join(pm.To, ", "))
	if len(pm.Cc) > 0 {
		fmt.Printf("Cc:         %s\n", strings.Join(pm.Cc, ", "))
	}
	fmt.Printf("Date:       %s\n", pm.Date)
	if pm.MessageID != "" {
		fmt.Printf("Message-ID: %s\n", pm.MessageID)
	}
	fmt.Println("\n----------")
	body := pm.TextBody
	if body == "" {
		body = pm.HTMLBody
	}
	fmt.Println(body)
	fmt.Println("----------")
	if len(pm.Attachments) > 0 {
		fmt.Println("Attachments:")
		for _, a := range pm.Attachments {
			fmt.Printf("  - %s (%s, %d bytes)\n", a.Filename, a.MimeType, a.Size)
		}
	}
}

func init() {
	rootCmd.AddCommand(readCmd)
	readCmd.Flags().StringVar(&readSaveAttachments, "save-attachments", "", "directory to save attachments into")
}
