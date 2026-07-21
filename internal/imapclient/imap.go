// Package imapclient connects to an IMAP server to read mail.
//
// Only implicit-TLS connections (port 993) are supported, which covers all
// built-in presets. Reading uses BODY.PEEK so messages are not marked as seen.
package imapclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"

	"MisakaMailClient/internal/provider"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Conn is an authenticated IMAP connection.
type Conn struct {
	c *client.Client
}

// Dial connects to the server and authenticates.
func Dial(server provider.Server, email, password string) (*Conn, error) {
	addr := net.JoinHostPort(server.Host, strconv.Itoa(server.Port))
	var c *client.Client
	var err error
	if server.SSL {
		c, err = client.DialTLS(addr, &tls.Config{ServerName: server.Host})
	} else {
		c, err = client.Dial(addr)
	}
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", addr, err)
	}
	if err := c.Login(email, password); err != nil {
		c.Logout()
		return nil, fmt.Errorf("login: %w", err)
	}
	return &Conn{c: c}, nil
}

// Close logs out and closes the connection.
func (c *Conn) Close() {
	if c != nil && c.c != nil {
		c.c.Logout()
	}
}

// Envelope is a summary of a message in the inbox.
type Envelope struct {
	Seq            uint32 `json:"seq"`
	UID            uint32 `json:"uid,omitempty"`
	Subject        string `json:"subject"`
	From           string `json:"from"`
	FromName       string `json:"from_name,omitempty"`
	Date           string `json:"date"`
	Seen           bool   `json:"seen"`
	HasAttachments bool   `json:"has_attachments"`
}

// Inbox lists the most recent messages in INBOX, newest first. limit<=0 means
// all messages. If unreadOnly is true, only unseen messages are returned.
func (c *Conn) Inbox(limit int, unreadOnly bool) ([]Envelope, error) {
	mbox, err := c.c.Select("INBOX", true)
	if err != nil {
		return nil, fmt.Errorf("select INBOX: %w", err)
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

	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchFlags,
		imap.FetchInternalDate,
		imap.FetchUid,
		imap.FetchBodyStructure,
	}

	msgs := make(chan *imap.Message, 100)
	done := make(chan error, 1)
	go func() {
		done <- c.c.Fetch(seqset, items, msgs)
	}()

	var out []Envelope
	for m := range msgs {
		if m == nil {
			continue
		}
		seen := false
		for _, f := range m.Flags {
			if f == imap.SeenFlag {
				seen = true
			}
		}
		if unreadOnly && seen {
			continue
		}
		e := Envelope{
			Seq:            m.SeqNum,
			UID:            m.Uid,
			Seen:           seen,
			HasAttachments: detectAttachments(m.BodyStructure),
		}
		if m.Envelope != nil {
			e.Subject = m.Envelope.Subject
			if len(m.Envelope.From) > 0 {
				e.From = m.Envelope.From[0].Address()
				e.FromName = m.Envelope.From[0].PersonalName
			}
			if !m.Envelope.Date.IsZero() {
				e.Date = m.Envelope.Date.Format("2006-01-02 15:04:05 -0700")
			}
		}
		if e.Date == "" && !m.InternalDate.IsZero() {
			e.Date = m.InternalDate.Format("2006-01-02 15:04:05 -0700")
		}
		out = append(out, e)
	}
	if err := <-done; err != nil {
		return out, fmt.Errorf("fetch: %w", err)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Seq > out[j].Seq })
	return out, nil
}

// detectAttachments reports whether the body structure contains any attachment
// parts (disposition "attachment", or a non-text part carrying a filename).
func detectAttachments(bs *imap.BodyStructure) bool {
	if bs == nil {
		return false
	}
	found := false
	bs.Walk(func(_ []int, part *imap.BodyStructure) bool {
		if part.Disposition == "attachment" {
			found = true
			return false
		}
		if part.MIMEType != "text" && part.MIMEType != "multipart" {
			if fn, _ := part.Filename(); fn != "" {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// FetchRaw returns the full RFC 822 bytes of the message with the given
// sequence number, without marking it as seen.
func (c *Conn) FetchRaw(seq uint32) ([]byte, error) {
	if _, err := c.c.Select("INBOX", true); err != nil {
		return nil, fmt.Errorf("select INBOX: %w", err)
	}
	if seq == 0 {
		return nil, fmt.Errorf("invalid sequence number 0")
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(seq)

	section := &imap.BodySectionName{Peek: true}
	items := []imap.FetchItem{section.FetchItem()}

	msgs := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.c.Fetch(seqset, items, msgs)
	}()

	var raw []byte
	for m := range msgs {
		if raw != nil || m == nil {
			continue
		}
		lit := m.GetBody(section)
		if lit != nil {
			raw, _ = io.ReadAll(lit)
		}
	}
	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch message %d: %w", seq, err)
	}
	if raw == nil {
		return nil, fmt.Errorf("message %d not found", seq)
	}
	return raw, nil
}

// FetchEnvelope returns envelope-level metadata for a single message, used to
// build reply headers without downloading the whole body.
func (c *Conn) FetchEnvelope(seq uint32) (*imap.Envelope, error) {
	if _, err := c.c.Select("INBOX", true); err != nil {
		return nil, fmt.Errorf("select INBOX: %w", err)
	}
	seqset := new(imap.SeqSet)
	seqset.AddNum(seq)
	items := []imap.FetchItem{imap.FetchEnvelope}

	msgs := make(chan *imap.Message, 1)
	done := make(chan error, 1)
	go func() {
		done <- c.c.Fetch(seqset, items, msgs)
	}()

	var env *imap.Envelope
	for m := range msgs {
		if m != nil {
			env = m.Envelope
		}
	}
	if err := <-done; err != nil {
		return nil, fmt.Errorf("fetch envelope %d: %w", seq, err)
	}
	if env == nil {
		return nil, fmt.Errorf("message %d not found", seq)
	}
	return env, nil
}
