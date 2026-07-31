package message

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
)

// cnWeekdays maps time.Weekday (Sunday=0) to the short Chinese weekday used in
// the reply-quote header, matching the Outlook/Aliyun Chinese-webmail style.
var cnWeekdays = [...]string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

// quoteSeparator is the dashed line drawn above the quoted-original header.
const quoteSeparator = "------------------------------------------------------------------"

// formatCNDate formats t as "2026年7月30日(周四) 10:37". It returns "" for a
// zero time so the 发送时间 line can be omitted when the date is unknown.
func formatCNDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return fmt.Sprintf("%d年%d月%d日(%s) %02d:%02d",
		t.Year(), int(t.Month()), t.Day(), cnWeekdays[t.Weekday()], t.Hour(), t.Minute())
}

// formatAddr renders one address as "Name <email>" when a display name is
// present, otherwise just the email.
func formatAddr(a *mail.Address) string {
	if a == nil {
		return ""
	}
	if name := strings.TrimSpace(a.Name); name != "" {
		return fmt.Sprintf("%s <%s>", name, a.Address)
	}
	return a.Address
}

// formatAddrList joins formatted addresses with "; ".
func formatAddrList(addrs []*mail.Address) string {
	parts := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if s := formatAddr(a); s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "; ")
}

// quoteHeaderLines returns the ordered header lines (without the separator) for
// the quoted original: 发件人 / 发送时间 / 收件人 / [抄　送] / 主　题. The 抄　送
// line is omitted when there are no Cc recipients.
func quoteHeaderLines(orig *ParsedMessage) []string {
	from := orig.From
	if strings.TrimSpace(orig.FromName) != "" {
		from = fmt.Sprintf("%s <%s>", orig.FromName, orig.From)
	}
	lines := []string{"发件人：" + from}

	if d := formatCNDate(orig.date); d != "" {
		lines = append(lines, "发送时间："+d)
	} else if orig.Date != "" {
		lines = append(lines, "发送时间："+orig.Date)
	}

	if s := formatAddrList(orig.toAddrs); s != "" {
		lines = append(lines, "收件人："+s)
	} else if len(orig.To) > 0 {
		lines = append(lines, "收件人："+strings.Join(orig.To, "; "))
	}

	if s := formatAddrList(orig.ccAddrs); s != "" {
		lines = append(lines, "抄　送："+s)
	} else if len(orig.Cc) > 0 {
		lines = append(lines, "抄　送："+strings.Join(orig.Cc, "; "))
	}

	lines = append(lines, "主　题："+orig.Subject)
	return lines
}

// QuoteText returns the original message formatted as a plain-text quote block
// to append below the reply body: the dashed separator, the header lines, and
// the original text body. When the original has no text part, the HTML body is
// reduced to text as a fallback.
func QuoteText(orig *ParsedMessage) string {
	var b strings.Builder
	b.WriteString(quoteSeparator)
	b.WriteByte('\n')
	for _, l := range quoteHeaderLines(orig) {
		b.WriteString(l)
		b.WriteByte('\n')
	}
	body := orig.TextBody
	if strings.TrimSpace(body) == "" {
		body = stripTags(orig.HTMLBody)
	}
	if body != "" {
		b.WriteByte('\n')
		b.WriteString(body)
	}
	return b.String()
}

// QuoteHTML returns the original message formatted as an HTML quote block that
// mail clients recognize as a collapsible quote. It mirrors the Aliyun webmail
// reply structure: a <blockquote> wrapping a <div class="alimail-quote"> that
// holds the dashed separator, the header lines, and the original body. The
// alimail-quote class is what the Aliyun web client keys on to show its
// "hide/show quoted content" toggle; other clients fall back to collapsing the
// <blockquote>. The original HTML has its outer html/head/body wrappers stripped
// so it nests cleanly.
func QuoteHTML(orig *ParsedMessage) string {
	var b strings.Builder
	b.WriteString(`<br><blockquote style="margin:0;padding:0;margin-left:1em;border-left:2px solid #bbb;color:#555;font-style:normal;">`)
	b.WriteString(`<div class="alimail-quote">`)
	b.WriteString(`<div>`)
	b.WriteString(html.EscapeString(quoteSeparator))
	b.WriteString(`</div>`)
	for _, l := range quoteHeaderLines(orig) {
		b.WriteString(`<div>`)
		b.WriteString(html.EscapeString(l))
		b.WriteString(`</div>`)
	}
	b.WriteString(`<div><br></div>`)
	body := stripHTMLDocument(orig.HTMLBody)
	if strings.TrimSpace(body) == "" {
		body = html.EscapeString(orig.TextBody)
	}
	if body != "" {
		b.WriteString(`<div>`)
		b.WriteString(body)
		b.WriteString(`</div>`)
	}
	b.WriteString(`</div></blockquote>`)
	return b.String()
}

// AppendQuote appends the quoted original to the reply body parts. The text
// quote is appended after a blank line; the HTML quote is inserted before
// </body> when the reply HTML is a full document, otherwise appended. Empty
// body parts are left untouched so Build still emits only the parts the caller
// provided.
func AppendQuote(textBody, htmlBody string, orig *ParsedMessage) (string, string) {
	if textBody != "" {
		textBody = textBody + "\n\n" + QuoteText(orig)
	}
	if htmlBody != "" {
		htmlBody = insertHTMLQuote(htmlBody, QuoteHTML(orig))
	}
	return textBody, htmlBody
}

// insertHTMLQuote inserts the quote markup before </body> (case-insensitive)
// when the HTML is a full document, otherwise appends it to the fragment.
func insertHTMLQuote(htmlBody, quote string) string {
	if idx := strings.LastIndex(strings.ToLower(htmlBody), "</body>"); idx >= 0 {
		return htmlBody[:idx] + quote + htmlBody[idx:]
	}
	return htmlBody + quote
}

var (
	reDoctype  = regexp.MustCompile(`(?i)<!doctype[^>]*>`)
	reHTMLHead = regexp.MustCompile(`(?is)<head\b[^>]*>.*?</head>`)
	reHTMLWrap = regexp.MustCompile(`(?is)</?html\b[^>]*>|</?body\b[^>]*>`)
	reHTMLTag  = regexp.MustCompile(`<[^>]*>`)
	reSpace    = regexp.MustCompile(`[ \t\r\n]+`)
)

// stripHTMLDocument removes the DOCTYPE, <head>...</head>, and outer
// html/body wrapper tags so the original HTML can be nested inside a quote div
// without producing a malformed document.
func stripHTMLDocument(s string) string {
	s = reDoctype.ReplaceAllString(s, "")
	s = reHTMLHead.ReplaceAllString(s, "")
	s = reHTMLWrap.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// stripTags reduces an HTML fragment to its text content, used as a fallback
// when quoting an original that has no plain-text part.
func stripTags(s string) string {
	s = reHTMLHead.ReplaceAllString(s, "")
	s = reHTMLTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = reSpace.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
