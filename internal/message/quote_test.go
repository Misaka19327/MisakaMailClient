package message

import (
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-message/mail"
)

func testQuoteMsg() *ParsedMessage {
	return &ParsedMessage{
		Subject:  "原主题 Original",
		From:     "wanghaisheng@hzhfzx.com",
		FromName: "王海生",
		To:       []string{"liuyixin@hzhfzx.com"},
		Cc:       []string{"dev_php@hzhfzx.com", "dev_product@hzhfzx.com"},
		Date:     "2026-07-30 10:37:00 +0800",
		TextBody: "原邮件正文第一行\n第二行",
		HTMLBody: "<div>原邮件正文</div>",
		date:     time.Date(2026, time.July, 30, 10, 37, 0, 0, time.FixedZone("CST", 8*3600)),
		toAddrs: []*mail.Address{
			{Name: "刘益鑫", Address: "liuyixin@hzhfzx.com"},
		},
		ccAddrs: []*mail.Address{
			{Address: "dev_php@hzhfzx.com"},
			{Address: "dev_product@hzhfzx.com"},
		},
	}
}

func TestQuoteText(t *testing.T) {
	orig := testQuoteMsg()
	q := QuoteText(orig)
	if !strings.HasPrefix(q, quoteSeparator) {
		t.Errorf("quote should start with separator: %q", q)
	}
	for _, want := range []string{
		"发件人：王海生 <wanghaisheng@hzhfzx.com>",
		"发送时间：2026年7月30日(周四) 10:37",
		"收件人：刘益鑫 <liuyixin@hzhfzx.com>",
		"抄　送：dev_php@hzhfzx.com; dev_product@hzhfzx.com",
		"主　题：原主题 Original",
		"原邮件正文第一行\n第二行",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("quote missing %q\nGot:\n%s", want, q)
		}
	}
}

func TestQuoteTextNoCc(t *testing.T) {
	orig := testQuoteMsg()
	orig.ccAddrs = nil
	orig.Cc = nil
	q := QuoteText(orig)
	if strings.Contains(q, "抄　送") {
		t.Errorf("quote should omit Cc line when none present: %q", q)
	}
}

func TestQuoteHTML(t *testing.T) {
	orig := testQuoteMsg()
	q := QuoteHTML(orig)
	// The quote is a <blockquote> wrapping <div class="alimail-quote">, the
	// structure the Aliyun web client keys on for its hide/show toggle.
	if !strings.Contains(q, `<blockquote`) {
		t.Errorf("HTML quote should be wrapped in <blockquote>: %q", q)
	}
	if !strings.Contains(q, `class="alimail-quote"`) {
		t.Errorf("HTML quote missing alimail-quote class: %q", q)
	}
	// Header lines are HTML-escaped and each wrapped in its own <div>.
	for _, want := range []string{
		"<div>发件人：王海生 &lt;wanghaisheng@hzhfzx.com&gt;</div>",
		"<div>发送时间：2026年7月30日(周四) 10:37</div>",
		"<div>收件人：刘益鑫 &lt;liuyixin@hzhfzx.com&gt;</div>",
		"<div>主　题：原主题 Original</div>",
		"<div>原邮件正文</div>",
	} {
		if !strings.Contains(q, want) {
			t.Errorf("HTML quote missing %q\nGot:\n%s", want, q)
		}
	}

	// A full-document original must have html/head/body wrappers stripped so it
	// nests cleanly inside the quote div.
	orig.HTMLBody = "<!DOCTYPE html><html><head><title>x</title></head><body><p>hi</p></body></html>"
	q2 := QuoteHTML(orig)
	if strings.Contains(q2, "<html") || strings.Contains(q2, "<head") || strings.Contains(q2, "<body") {
		t.Errorf("HTML quote did not strip document wrappers: %q", q2)
	}
	if !strings.Contains(q2, "<p>hi</p>") {
		t.Errorf("HTML quote lost inner content: %q", q2)
	}
}

func TestAppendQuote(t *testing.T) {
	orig := testQuoteMsg()
	text, html := AppendQuote("reply body", "<html><body><p>reply</p></body></html>", orig)
	if !strings.HasPrefix(text, "reply body\n\n") {
		t.Errorf("text quote should follow reply body after a blank line: %q", text)
	}
	if !strings.Contains(text, "发件人：") {
		t.Errorf("text quote missing header: %q", text)
	}
	// HTML quote is inserted after the reply body, before </body>, as a
	// collapsible <blockquote>.
	if !strings.Contains(html, "<p>reply</p><br><blockquote") {
		t.Errorf("HTML quote not inserted before </body>: %q", html)
	}
	if !strings.HasSuffix(strings.TrimSpace(html), "</blockquote></body></html>") {
		t.Errorf("HTML tail not preserved: %q", html)
	}

	// Empty body parts are left untouched.
	text2, html2 := AppendQuote("", "", orig)
	if text2 != "" || html2 != "" {
		t.Errorf("empty bodies should stay empty: %q / %q", text2, html2)
	}
}

func TestFormatCNDate(t *testing.T) {
	tm := time.Date(2026, time.July, 30, 10, 37, 0, 0, time.FixedZone("CST", 8*3600))
	if got := formatCNDate(tm); got != "2026年7月30日(周四) 10:37" {
		t.Errorf("formatCNDate: got %q want %q", got, "2026年7月30日(周四) 10:37")
	}
	if got := formatCNDate(time.Time{}); got != "" {
		t.Errorf("formatCNDate zero time: got %q want empty", got)
	}
}
