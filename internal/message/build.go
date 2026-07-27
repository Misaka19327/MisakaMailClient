// Package message builds and parses RFC 822 mail messages.
package message

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
)

// DefaultFooterText is the notice appended to the end of outgoing mail by
// default, when no --footer override is given.
const DefaultFooterText = "本邮件由 misaka-mail CLI 辅助发送"

// SendSpec describes a message to build.
type SendSpec struct {
	From     string   // sender email address
	FromName string   // sender display name (optional)
	To       []string // To recipients
	Cc       []string // Cc recipients
	Bcc      []string // Bcc recipients (envelope only, never written to headers)
	Subject  string
	TextBody string // plain-text body (optional)
	HTMLBody string // HTML body (optional)
	// Footer is a notice appended to the end of the body in a fixed,
	// non-configurable style: a parenthesized line in the plain-text part and a
	// small gray <small> in the HTML part. Empty means no footer.
	Footer      string
	Attachments []string // attachment file paths
	// InReplyTo is the Message-ID being replied to, without angle brackets.
	InReplyTo string
	// References is the References chain, without angle brackets.
	References []string
}

// AllRecipients returns To + Cc + Bcc for use as the SMTP envelope recipients.
func (s SendSpec) AllRecipients() []string {
	out := make([]string, 0, len(s.To)+len(s.Cc)+len(s.Bcc))
	out = append(out, s.To...)
	out = append(out, s.Cc...)
	out = append(out, s.Bcc...)
	return out
}

func addrList(addrs []string) []*mail.Address {
	out := make([]*mail.Address, 0, len(addrs))
	for _, a := range addrs {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		out = append(out, &mail.Address{Address: a})
	}
	return out
}

// Build renders the message as RFC 822 bytes and returns the generated
// Message-ID (without angle brackets).
func Build(spec SendSpec) (raw []byte, messageID string, err error) {
	if spec.From == "" {
		return nil, "", fmt.Errorf("from address is required")
	}
	if len(spec.To)+len(spec.Cc)+len(spec.Bcc) == 0 {
		return nil, "", fmt.Errorf("at least one recipient (to/cc/bcc) is required")
	}

	var h mail.Header
	from := &mail.Address{Name: spec.FromName, Address: spec.From}
	h.SetAddressList("From", []*mail.Address{from})
	h.SetAddressList("To", addrList(spec.To))
	h.SetAddressList("Cc", addrList(spec.Cc))
	// Bcc is intentionally NOT written to the headers; it only appears in the
	// SMTP envelope so recipients cannot see each other.
	h.SetSubject(spec.Subject)
	h.SetDate(time.Now())
	if err := h.GenerateMessageIDWithHostname(hostOf(spec.From)); err != nil {
		return nil, "", fmt.Errorf("generate message-id: %w", err)
	}
	messageID, _ = h.MessageID()
	if spec.InReplyTo != "" {
		h.SetMsgIDList("In-Reply-To", []string{spec.InReplyTo})
	}
	if len(spec.References) > 0 {
		h.SetMsgIDList("References", spec.References)
	}

	var buf bytes.Buffer
	mw, err := mail.CreateWriter(&buf, h)
	if err != nil {
		return nil, "", fmt.Errorf("create message writer: %w", err)
	}

	// Apply the footer notice (fixed style) to whichever body parts exist.
	textBody := spec.TextBody
	htmlBody := spec.HTMLBody
	if spec.Footer != "" {
		if textBody != "" {
			textBody = textBody + "\n（" + spec.Footer + "）"
		}
		if htmlBody != "" {
			htmlBody = insertHTMLFooter(htmlBody, html.EscapeString(spec.Footer))
		}
	}

	// Inline body (text and/or html) as multipart/alternative.
	if textBody != "" || htmlBody != "" {
		iw, err := mw.CreateInline()
		if err != nil {
			mw.Close()
			return nil, "", fmt.Errorf("create inline writer: %w", err)
		}
		if textBody != "" {
			var ih mail.InlineHeader
			ih.Set("Content-Type", "text/plain; charset=utf-8")
			pw, err := iw.CreatePart(ih)
			if err != nil {
				iw.Close()
				mw.Close()
				return nil, "", fmt.Errorf("create text part: %w", err)
			}
			_, _ = pw.Write([]byte(textBody))
			pw.Close()
		}
		if htmlBody != "" {
			var ih mail.InlineHeader
			ih.Set("Content-Type", "text/html; charset=utf-8")
			pw, err := iw.CreatePart(ih)
			if err != nil {
				iw.Close()
				mw.Close()
				return nil, "", fmt.Errorf("create html part: %w", err)
			}
			_, _ = pw.Write([]byte(htmlBody))
			pw.Close()
		}
		iw.Close()
	}

	// Attachments.
	for _, p := range spec.Attachments {
		data, err := os.ReadFile(p)
		if err != nil {
			mw.Close()
			return nil, "", fmt.Errorf("read attachment %s: %w", p, err)
		}
		var ah mail.AttachmentHeader
		ah.Set("Content-Type", mimeTypeFor(p))
		ah.SetFilename(filepath.Base(p))
		aw, err := mw.CreateAttachment(ah)
		if err != nil {
			mw.Close()
			return nil, "", fmt.Errorf("create attachment %s: %w", p, err)
		}
		_, _ = aw.Write(data)
		aw.Close()
	}

	if err := mw.Close(); err != nil {
		return nil, "", fmt.Errorf("finalize message: %w", err)
	}
	return buf.Bytes(), messageID, nil
}

func hostOf(email string) string {
	if i := strings.LastIndex(email, "@"); i >= 0 {
		return email[i+1:]
	}
	return "localhost"
}

// insertHTMLFooter inserts the footer markup before </body> (case-insensitive)
// when the HTML is a full document, otherwise appends it to the fragment. The
// footer text must already be HTML-escaped. The wrapping style is fixed.
func insertHTMLFooter(htmlBody, escapedFooter string) string {
	footer := "<br><small style=\"color:#888\">" + escapedFooter + "</small>"
	if idx := strings.LastIndex(strings.ToLower(htmlBody), "</body>"); idx >= 0 {
		return htmlBody[:idx] + footer + htmlBody[idx:]
	}
	return htmlBody + footer
}

func mimeTypeFor(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt":
		return "text/plain"
	case ".html", ".htm":
		return "text/html"
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		return "application/octet-stream"
	}
}
