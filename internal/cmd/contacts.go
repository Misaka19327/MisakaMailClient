package cmd

import (
	"fmt"
	"time"

	"MisakaMailClient/internal/config"
	"MisakaMailClient/internal/contacts"
	"MisakaMailClient/internal/crypto"
	"MisakaMailClient/internal/imapclient"
	"MisakaMailClient/internal/output"

	"github.com/spf13/cobra"
)

var (
	contactsLimit   int
	contactsInbox   bool
	contactsRefresh bool
	contactsMerge   bool
)

var contactsCmd = &cobra.Command{
	Use:   "contacts",
	Short: "Pull contacts from the mailbox (cached, encrypted locally)",
	Long: "Pull contacts from the mailbox. Tries a Contacts folder (vCard)\n" +
		"first, then scans the Sent folder for recipients (portable fallback,\n" +
		"since IMAP has no standard address-book API). --include-inbox also\n" +
		"collects senders from INBOX.\n\n" +
		"Results are cached locally, encrypted with the app encryption key\n" +
		"(set via 'misaka-mail log key'). By default the cached list is returned;\n" +
		"use --refresh to re-pull and replace the cache, or --merge to re-pull and\n" +
		"merge with the cache. Without a key, contacts are pulled but not cached.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if contactsRefresh && contactsMerge {
			return fmt.Errorf("--refresh and --merge are mutually exclusive")
		}
		acc, err := resolveAccount()
		if err != nil {
			return err
		}
		pwd, err := accountPassword(acc)
		if err != nil {
			return err
		}
		opts := contacts.Options{Limit: contactsLimit, IncludeInbox: contactsInbox}
		hasKey := crypto.HasKey()

		// Default: serve from cache when available.
		if !contactsRefresh && !contactsMerge {
			if hasKey {
				if cache, err := contacts.Load(acc.Email); err == nil && cache != nil {
					return emitContacts(acc.Email, cache.Contacts, cache.PulledAt, "cache", nil)
				}
			}
			list, notes, pulledAt, err := pullAndCache(acc, pwd, opts, hasKey)
			if err != nil {
				return err
			}
			return emitContacts(acc.Email, list, pulledAt, "server", notes)
		}

		// --refresh: replace cache with a fresh pull.
		if contactsRefresh {
			list, notes, pulledAt, err := pullAndCache(acc, pwd, opts, hasKey)
			if err != nil {
				return err
			}
			return emitContacts(acc.Email, list, pulledAt, "refresh", notes)
		}

		// --merge: pull and merge with the existing cache.
		conn, err := imapclient.Dial(acc.IMAP, acc.Email, pwd)
		if err != nil {
			return err
		}
		defer conn.Close()
		fresh, notes, err := contacts.Pull(conn, opts)
		if err != nil {
			return err
		}
		if !hasKey {
			notes = append(notes, "no encryption key set; not cached")
			return emitContacts(acc.Email, fresh, time.Time{}, "server", notes)
		}
		var existing []contacts.Contact
		if cache, err := contacts.Load(acc.Email); err == nil && cache != nil {
			existing = cache.Contacts
			notes = append(notes, "merged with existing cache")
		}
		merged := contacts.Merge(existing, fresh)
		if err := contacts.Save(acc.Email, merged); err != nil {
			return fmt.Errorf("save merged contacts: %w", err)
		}
		return emitContacts(acc.Email, merged, time.Now(), "merge", notes)
	},
}

// pullAndCache dials IMAP, pulls contacts, and saves them (when a key is set).
func pullAndCache(acc *config.Account, pwd string, opts contacts.Options, hasKey bool) (list []contacts.Contact, notes []string, pulledAt time.Time, err error) {
	conn, err := imapclient.Dial(acc.IMAP, acc.Email, pwd)
	if err != nil {
		return nil, nil, time.Time{}, err
	}
	defer conn.Close()
	list, notes, err = contacts.Pull(conn, opts)
	if err != nil {
		return nil, nil, time.Time{}, err
	}
	pulledAt = time.Now()
	if hasKey {
		if e := contacts.Save(acc.Email, list); e != nil {
			notes = append(notes, "cache save failed: "+e.Error())
		} else {
			notes = append(notes, "cached locally (encrypted)")
		}
	} else {
		notes = append(notes, "no encryption key set; not cached (run 'misaka-mail log key')")
	}
	return list, notes, pulledAt, nil
}

func emitContacts(account string, list []contacts.Contact, pulledAt time.Time, source string, notes []string) error {
	if jsonMode {
		m := map[string]interface{}{
			"account":  account,
			"count":    len(list),
			"contacts": list,
			"source":   source,
			"notes":    notes,
		}
		if !pulledAt.IsZero() {
			m["pulled_at"] = pulledAt.Format("2006-01-02 15:04:05")
		}
		return output.PrintJSON(m)
	}
	fmt.Printf("Contacts for %s (%d) [source: %s]\n", account, len(list), source)
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
}

func init() {
	rootCmd.AddCommand(contactsCmd)
	contactsCmd.Flags().IntVar(&contactsLimit, "limit", 200, "max messages to scan per folder (0 = all)")
	contactsCmd.Flags().BoolVar(&contactsInbox, "include-inbox", false, "also collect senders from INBOX")
	contactsCmd.Flags().BoolVar(&contactsRefresh, "refresh", false, "force re-pull from server and replace the local cache")
	contactsCmd.Flags().BoolVar(&contactsMerge, "merge", false, "force re-pull from server and merge with the local cache")
}
