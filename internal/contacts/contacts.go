// Package contacts pulls a contact list from an IMAP account: it tries a
// Contacts folder (vCard) first, then scans the Sent folder (and optionally
// INBOX) envelopes for addresses.
package contacts

import (
	"sort"
	"strconv"
	"strings"

	"MisakaMailClient/internal/imapclient"
	"MisakaMailClient/internal/vcard"
)

// Contact is a deduplicated contact.
type Contact struct {
	Name   string `json:"name"`
	Email  string `json:"email"`
	Source string `json:"source"` // "vcard", "sent", "inbox"
}

// Options controls the pull behavior.
type Options struct {
	Limit        int  // max messages to scan per folder (0 = all)
	IncludeInbox bool // also scan INBOX senders
}

// Pull fetches contacts and returns them with informational notes about what
// was tried (e.g. which folders were found).
func Pull(conn *imapclient.Conn, opts Options) (list []Contact, notes []string, err error) {
	mboxes, err := conn.ListMailboxes()
	if err != nil {
		return nil, nil, err
	}

	byEmail := map[string]*Contact{}
	add := func(name, email, source string) {
		email = strings.TrimSpace(strings.ToLower(email))
		if email == "" || !strings.Contains(email, "@") {
			return
		}
		name = strings.TrimSpace(name)
		if ex, ok := byEmail[email]; ok {
			if ex.Name == "" && name != "" {
				ex.Name = name
			}
			return
		}
		byEmail[email] = &Contact{Name: name, Email: email, Source: source}
	}

	// 1. Contacts folder (vCard).
	if cf := imapclient.FindContacts(mboxes); cf.Name != "" {
		notes = append(notes, "contacts folder: "+cf.Display)
		raw, ferr := conn.FetchVCards(cf.Name, opts.Limit)
		if ferr == nil {
			count := 0
			for _, r := range raw {
				for _, card := range vcard.Parse(r) {
					for _, e := range card.Emails {
						add(card.Name, e, "vcard")
						count++
					}
				}
			}
			notes = append(notes, "vcard emails parsed: "+strconv.Itoa(count))
		} else {
			notes = append(notes, "vcard fetch failed: "+ferr.Error())
		}
	} else {
		notes = append(notes, "no contacts folder found")
	}

	// 2. Sent folder envelopes.
	if sf := imapclient.FindSent(mboxes); sf.Name != "" {
		notes = append(notes, "sent folder: "+sf.Display)
		addrs, ferr := conn.CollectFromEnvelopes(sf.Name, opts.Limit)
		if ferr == nil {
			for _, a := range addrs {
				add(a.Name, a.Email, "sent")
			}
			notes = append(notes, "sent messages scanned, addresses: "+strconv.Itoa(len(addrs)))
		} else {
			notes = append(notes, "sent scan failed: "+ferr.Error())
		}
	} else {
		notes = append(notes, "no sent folder found")
	}

	// 3. Optional INBOX senders.
	if opts.IncludeInbox {
		addrs, ferr := conn.CollectFromEnvelopes("INBOX", opts.Limit)
		if ferr == nil {
			for _, a := range addrs {
				add(a.Name, a.Email, "inbox")
			}
			notes = append(notes, "inbox messages scanned, addresses: "+strconv.Itoa(len(addrs)))
		} else {
			notes = append(notes, "inbox scan failed: "+ferr.Error())
		}
	}

	list = make([]Contact, 0, len(byEmail))
	for _, c := range byEmail {
		list = append(list, *c)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Name != list[j].Name {
			return list[i].Name < list[j].Name
		}
		return list[i].Email < list[j].Email
	})
	return list, notes, nil
}
