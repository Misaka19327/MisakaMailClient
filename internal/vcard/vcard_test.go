package vcard

import "testing"

func TestParse(t *testing.T) {
	data := "BEGIN:VCARD\r\nVERSION:3.0\r\nFN:Alice 张\r\nN:张;Alice;;;\r\nEMAIL:alice@example.com\r\nEMAIL:alice@work.com\r\nEND:VCARD\r\nBEGIN:VCARD\r\nFN:Bob\r\nEMAIL:bob@example.com\r\nEND:VCARD\r\n"
	cards := Parse(data)
	if len(cards) != 2 {
		t.Fatalf("got %d cards, want 2", len(cards))
	}
	if cards[0].Name != "Alice 张" {
		t.Errorf("card0 name: got %q want %q", cards[0].Name, "Alice 张")
	}
	if len(cards[0].Emails) != 2 {
		t.Errorf("card0 emails: got %v", cards[0].Emails)
	}
	if cards[1].Name != "Bob" || cards[1].Emails[0] != "bob@example.com" {
		t.Errorf("card1: %+v", cards[1])
	}
}

func TestParseFoldedLine(t *testing.T) {
	// RFC 6350 folding: CRLF + a single WSP is removed on unfold. The original
	// space stays at the end of the first line; the continuation's leading WSP
	// is the fold marker and is dropped.
	data := "BEGIN:VCARD\r\nFN:Alice \r\n Smith\r\nEMAIL:a@b.com\r\nEND:VCARD\r\n"
	cards := Parse(data)
	if len(cards) != 1 {
		t.Fatalf("got %d cards, want 1", len(cards))
	}
	if cards[0].Name != "Alice Smith" {
		t.Errorf("folded name: got %q want %q", cards[0].Name, "Alice Smith")
	}
	if cards[0].Emails[0] != "a@b.com" {
		t.Errorf("email: got %v", cards[0].Emails)
	}
}
