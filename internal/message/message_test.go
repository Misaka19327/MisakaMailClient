package message

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildParseRoundTrip builds a message with text + html + threading headers
// and verifies Parse recovers the same content.
func TestBuildParseRoundTrip(t *testing.T) {
	spec := SendSpec{
		From:      "alice@example.com",
		FromName:  "爱丽丝",
		To:        []string{"bob@example.com", "carol@example.com"},
		Cc:        []string{"dan@example.com"},
		Subject:   "测试主题 Test Subject",
		TextBody:  "你好,世界。\nPlain text body.",
		HTMLBody:  "<p>你好,<b>世界</b>。</p>",
		InReplyTo: "original@example.com",
		References: []string{"grandparent@example.com", "parent@example.com"},
	}
	raw, msgID, err := Build(spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if msgID == "" {
		t.Fatal("Build returned empty message ID")
	}

	pm, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if pm.Subject != spec.Subject {
		t.Errorf("Subject: got %q want %q", pm.Subject, spec.Subject)
	}
	if pm.From != spec.From {
		t.Errorf("From: got %q want %q", pm.From, spec.From)
	}
	if pm.FromName != spec.FromName {
		t.Errorf("FromName: got %q want %q", pm.FromName, spec.FromName)
	}
	if len(pm.To) != 2 || pm.To[0] != "bob@example.com" || pm.To[1] != "carol@example.com" {
		t.Errorf("To: got %v", pm.To)
	}
	if len(pm.Cc) != 1 || pm.Cc[0] != "dan@example.com" {
		t.Errorf("Cc: got %v", pm.Cc)
	}
	if pm.TextBody != spec.TextBody {
		t.Errorf("TextBody: got %q want %q", pm.TextBody, spec.TextBody)
	}
	if pm.HTMLBody != spec.HTMLBody {
		t.Errorf("HTMLBody: got %q want %q", pm.HTMLBody, spec.HTMLBody)
	}
	if pm.MessageID != msgID {
		t.Errorf("MessageID: got %q want %q", pm.MessageID, msgID)
	}
	if pm.InReplyTo != spec.InReplyTo {
		t.Errorf("InReplyTo: got %q want %q", pm.InReplyTo, spec.InReplyTo)
	}
	// References should include the original chain plus the parent message id.
	if len(pm.References) != 2 {
		t.Errorf("References: got %v", pm.References)
	}

	// Bcc must NOT appear in the serialized headers.
	if bytes.Contains(raw, []byte("Bcc")) {
		t.Error("Bcc header leaked into message")
	}
}

// TestBuildWithAttachment verifies attachment metadata round-trips through Parse.
func TestBuildWithAttachment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")
	payload := "attachment payload — 附件内容"
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	spec := SendSpec{
		From:        "alice@example.com",
		To:          []string{"bob@example.com"},
		Subject:     "with attachment",
		TextBody:     "see attached",
		Attachments: []string{path},
	}
	raw, _, err := Build(spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pm, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(pm.Attachments) != 1 {
		t.Fatalf("Attachments: got %d want 1", len(pm.Attachments))
	}
	a := pm.Attachments[0]
	if a.Filename != "report.txt" {
		t.Errorf("Filename: got %q want report.txt", a.Filename)
	}
	if a.Size != len(payload) {
		t.Errorf("Size: got %d want %d", a.Size, len(payload))
	}
	if !strings.HasPrefix(a.MimeType, "text/plain") {
		t.Errorf("MimeType: got %q", a.MimeType)
	}
}

// TestBuildValidation checks that Build rejects invalid specs.
func TestBuildValidation(t *testing.T) {
	if _, _, err := Build(SendSpec{To: []string{"x@example.com"}}); err == nil {
		t.Error("expected error for missing From")
	}
	if _, _, err := Build(SendSpec{From: "x@example.com"}); err == nil {
		t.Error("expected error for missing recipients")
	}
}

// TestBuildFooter verifies the footer notice is appended in the fixed style to
// both body parts and can be omitted by leaving Footer empty.
func TestBuildFooter(t *testing.T) {
	spec := SendSpec{
		From:     "alice@example.com",
		To:       []string{"bob@example.com"},
		Subject:  "footer test",
		TextBody: "hello",
		HTMLBody: "<div>hello</div>",
		Footer:   "sent by bot",
	}
	raw, _, err := Build(spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pm, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if want := "hello\n（sent by bot）"; pm.TextBody != want {
		t.Errorf("TextBody: got %q want %q", pm.TextBody, want)
	}
	if want := `<div>hello</div><br><small style="color:#888">sent by bot</small>`; pm.HTMLBody != want {
		t.Errorf("HTMLBody: got %q want %q", pm.HTMLBody, want)
	}

	// No footer when Footer is empty.
	spec.Footer = ""
	raw, _, err = Build(spec)
	if err != nil {
		t.Fatalf("Build (no footer): %v", err)
	}
	pm, err = Parse(raw)
	if err != nil {
		t.Fatalf("Parse (no footer): %v", err)
	}
	if strings.Contains(pm.TextBody, "sent by bot") {
		t.Errorf("TextBody should not have footer: %q", pm.TextBody)
	}
	if strings.Contains(pm.HTMLBody, "<small") {
		t.Errorf("HTMLBody should not have footer: %q", pm.HTMLBody)
	}
}

// TestBuildFooterEscapesHTML verifies the footer text is HTML-escaped so a
// custom footer cannot inject markup.
func TestBuildFooterEscapesHTML(t *testing.T) {
	spec := SendSpec{
		From:     "alice@example.com",
		To:       []string{"bob@example.com"},
		Subject:  "escape test",
		HTMLBody: "<div>hi</div>",
		Footer:   "<b>bold</b> & more",
	}
	raw, _, err := Build(spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pm, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !strings.Contains(pm.HTMLBody, "&lt;b&gt;bold&lt;/b&gt; &amp; more") {
		t.Errorf("HTMLBody footer not escaped: %q", pm.HTMLBody)
	}
	if strings.Contains(pm.HTMLBody, "<b>bold</b>") {
		t.Errorf("HTMLBody footer leaked raw markup: %q", pm.HTMLBody)
	}
}

// TestInsertHTMLFooterDocument verifies the footer is inserted before </body>
// for full-document HTML rather than appended after it.
func TestInsertHTMLFooterDocument(t *testing.T) {
	in := "<html><body><p>hi</p></body></html>"
	out := insertHTMLFooter(in, "notice")
	if !strings.Contains(out, "</p><br><small style=\"color:#888\">notice</small></body>") {
		t.Errorf("footer not inserted before </body>: %q", out)
	}
}
