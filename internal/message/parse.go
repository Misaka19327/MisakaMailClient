package message

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
)

// wordDecoder decodes RFC 2047 encoded-word headers (e.g. =?UTF-8?B?...?=).
var wordDecoder = &mime.WordDecoder{CharsetReader: message.CharsetReader}

func decodeWords(s string) string {
	if s == "" {
		return s
	}
	out, err := wordDecoder.DecodeHeader(s)
	if err != nil {
		return s
	}
	return out
}

// ParsedAttachment is metadata for a decoded attachment.
type ParsedAttachment struct {
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int    `json:"size"`
}

// ParsedMessage is a structured representation of a received mail.
type ParsedMessage struct {
	MessageID   string             `json:"message_id,omitempty"`
	InReplyTo   string             `json:"in_reply_to,omitempty"`
	References  []string           `json:"references,omitempty"`
	Subject     string             `json:"subject"`
	From        string             `json:"from"`
	FromName    string             `json:"from_name,omitempty"`
	To          []string           `json:"to"`
	Cc          []string           `json:"cc,omitempty"`
	Date        string             `json:"date"`
	TextBody    string             `json:"text_body"`
	HTMLBody    string             `json:"html_body,omitempty"`
	Attachments []ParsedAttachment `json:"attachments,omitempty"`
}

// SavedAttachment is an attachment written to disk.
type SavedAttachment struct {
	ParsedAttachment
	Path string `json:"path"`
}

func addrStrings(addrs []*mail.Address) []string {
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, a.Address)
	}
	return out
}

// Parse decodes raw RFC 822 bytes into a ParsedMessage. Attachment bodies are
// not retained (only metadata); use SaveAttachments to write them to disk.
func Parse(raw []byte) (*ParsedMessage, error) {
	r, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil && !message.IsUnknownCharset(err) {
		return nil, fmt.Errorf("parse message: %w", err)
	}
	defer r.Close()

	pm := &ParsedMessage{}
	h := r.Header
	pm.Subject, _ = h.Subject()
	if id, err := h.MessageID(); err == nil {
		pm.MessageID = id
	}
	if ids, err := h.MsgIDList("In-Reply-To"); err == nil && len(ids) > 0 {
		pm.InReplyTo = ids[0]
	}
	if ids, err := h.MsgIDList("References"); err == nil {
		pm.References = ids
	}
	if t, err := h.Date(); err == nil && !t.IsZero() {
		pm.Date = t.Format("2006-01-02 15:04:05 -0700")
	} else {
		pm.Date = strings.TrimSpace(h.Get("Date"))
	}
	if from, err := h.AddressList("From"); err == nil && len(from) > 0 {
		pm.From = from[0].Address
		pm.FromName = decodeWords(from[0].Name)
	}
	if to, err := h.AddressList("To"); err == nil {
		pm.To = addrStrings(to)
	}
	if cc, err := h.AddressList("Cc"); err == nil {
		pm.Cc = addrStrings(cc)
	}

	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil && !message.IsUnknownCharset(err) {
			return pm, fmt.Errorf("read part: %w", err)
		}
		body, _ := io.ReadAll(p.Body)
		switch hd := p.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := hd.ContentType()
			// Normalize CRLF to LF for cleaner display and JSON output.
			s := strings.ReplaceAll(string(body), "\r\n", "\n")
			if strings.HasPrefix(strings.ToLower(ct), "text/html") {
				pm.HTMLBody += s
			} else {
				pm.TextBody += s
			}
		case *mail.AttachmentHeader:
			fn, _ := hd.Filename()
			ct, _, _ := hd.ContentType()
			pm.Attachments = append(pm.Attachments, ParsedAttachment{
				Filename: fn,
				MimeType: ct,
				Size:     len(body),
			})
		}
	}
	return pm, nil
}

var unsafeFilename = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]`)

func sanitizeFilename(name string) string {
	name = unsafeFilename.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	if name == "" {
		name = "attachment"
	}
	return name
}

func uniquePath(p string) string {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	dir, file := filepath.Split(p)
	ext := filepath.Ext(file)
	base := strings.TrimSuffix(file, ext)
	for i := 1; ; i++ {
		cand := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(cand); os.IsNotExist(err) {
			return cand
		}
	}
}

// SaveAttachments parses raw and writes each attachment into dir, returning
// metadata including the on-disk path. Non-attachment parts are skipped.
func SaveAttachments(raw []byte, dir string) ([]SavedAttachment, error) {
	r, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil && !message.IsUnknownCharset(err) {
		return nil, fmt.Errorf("parse message: %w", err)
	}
	defer r.Close()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	var out []SavedAttachment
	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil && !message.IsUnknownCharset(err) {
			return out, fmt.Errorf("read part: %w", err)
		}
		ah, ok := p.Header.(*mail.AttachmentHeader)
		if !ok {
			_, _ = io.Copy(io.Discard, p.Body)
			continue
		}
		fn, _ := ah.Filename()
		ct, _, _ := ah.ContentType()
		data, _ := io.ReadAll(p.Body)
		path := uniquePath(filepath.Join(dir, sanitizeFilename(fn)))
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return out, err
		}
		out = append(out, SavedAttachment{
			ParsedAttachment: ParsedAttachment{Filename: fn, MimeType: ct, Size: len(data)},
			Path:             path,
		})
	}
	return out, nil
}
