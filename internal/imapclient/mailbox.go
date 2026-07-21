package imapclient

import (
	"io"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/utf7"
)

// Mailbox is a listed IMAP mailbox.
type Mailbox struct {
	Name       string   // raw name (for SELECT)
	Display    string   // decoded name (for display)
	Attributes []string // mailbox attributes / special-use flags
}

// Addr is a name + email pair extracted from a message envelope.
type Addr struct {
	Name  string
	Email string
}

func decodeName(s string) string {
	out, err := utf7.Encoding.NewDecoder().String(s)
	if err != nil || out == "" {
		return s
	}
	return out
}

// ListMailboxes lists all mailboxes in the account.
func (c *Conn) ListMailboxes() ([]Mailbox, error) {
	mboxes := make(chan *imap.MailboxInfo, 50)
	done := make(chan error, 1)
	go func() { done <- c.c.List("", "*", mboxes) }()
	var out []Mailbox
	for m := range mboxes {
		if m == nil {
			continue
		}
		out = append(out, Mailbox{
			Name:       m.Name,
			Display:    decodeName(m.Name),
			Attributes: m.Attributes,
		})
	}
	if err := <-done; err != nil {
		return nil, err
	}
	return out, nil
}

func hasAttr(m Mailbox, attr string) bool {
	for _, a := range m.Attributes {
		if strings.EqualFold(a, attr) {
			return true
		}
	}
	return false
}

// FindSent returns the Sent folder (by \Sent special-use flag, else by name),
// or a zero Mailbox if not found.
func FindSent(mboxes []Mailbox) Mailbox {
	for _, m := range mboxes {
		if hasAttr(m, "\\Sent") {
			return m
		}
	}
	for _, m := range mboxes {
		switch strings.ToLower(m.Display) {
		case "sent", "sent items", "sent messages", "已发送", "已发件箱", "发件箱":
			return m
		}
	}
	return Mailbox{}
}

// FindContacts returns a contacts-like folder by name, or a zero Mailbox if
// not found. There is no standard special-use flag for contacts.
func FindContacts(mboxes []Mailbox) Mailbox {
	for _, m := range mboxes {
		switch strings.ToLower(m.Display) {
		case "contacts", "联系人", "通讯录", "address book", "addressbook":
			return m
		}
	}
	return Mailbox{}
}

// FetchVCards fetches the raw body of up to limit messages in folder (newest
// first when limit applies). Bodies are returned as-is for vCard parsing.
func (c *Conn) FetchVCards(folder string, limit int) ([]string, error) {
	mbox, err := c.c.Select(folder, true)
	if err != nil {
		return nil, err
	}
	if mbox.Messages == 0 {
		return nil, nil
	}
	from := uint32(1)
	to := mbox.Messages
	if limit > 0 && uint32(limit) < mbox.Messages {
		from = mbox.Messages - uint32(limit) + 1
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	section := &imap.BodySectionName{Peek: true}
	items := []imap.FetchItem{section.FetchItem()}

	msgs := make(chan *imap.Message, 50)
	done := make(chan error, 1)
	go func() { done <- c.c.Fetch(seqset, items, msgs) }()

	var out []string
	for m := range msgs {
		if m == nil {
			continue
		}
		lit := m.GetBody(section)
		if lit == nil {
			continue
		}
		data, _ := io.ReadAll(lit)
		out = append(out, string(data))
	}
	if err := <-done; err != nil {
		return nil, err
	}
	return out, nil
}

// CollectFromEnvelopes fetches envelopes from folder and extracts all
// addresses (From/To/Cc). limit caps the number of messages scanned (newest).
func (c *Conn) CollectFromEnvelopes(folder string, limit int) ([]Addr, error) {
	mbox, err := c.c.Select(folder, true)
	if err != nil {
		return nil, err
	}
	if mbox.Messages == 0 {
		return nil, nil
	}
	from := uint32(1)
	to := mbox.Messages
	if limit > 0 && uint32(limit) < mbox.Messages {
		from = mbox.Messages - uint32(limit) + 1
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	items := []imap.FetchItem{imap.FetchEnvelope}
	msgs := make(chan *imap.Message, 100)
	done := make(chan error, 1)
	go func() { done <- c.c.Fetch(seqset, items, msgs) }()

	var out []Addr
	add := func(addrs []*imap.Address) {
		for _, a := range addrs {
			if a == nil {
				continue
			}
			out = append(out, Addr{Name: a.PersonalName, Email: a.Address()})
		}
	}
	for m := range msgs {
		if m == nil || m.Envelope == nil {
			continue
		}
		add(m.Envelope.From)
		add(m.Envelope.To)
		add(m.Envelope.Cc)
	}
	if err := <-done; err != nil {
		return nil, err
	}
	return out, nil
}
