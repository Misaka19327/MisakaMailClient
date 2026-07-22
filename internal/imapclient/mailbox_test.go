package imapclient

import "testing"

func TestResolveFolderName(t *testing.T) {
	// A realistic-ish mailbox list: raw names differ from decoded display
	// names, and the Sent folder carries the \Sent special-use flag.
	mboxes := []Mailbox{
		{Name: "INBOX", Display: "INBOX"},
		{Name: "SentRaw", Display: "已发送", Attributes: []string{"\\Sent"}},
		{Name: "Drafts", Display: "草稿箱"},
		{Name: "ContactsRaw", Display: "联系人"},
		{Name: "Trash", Display: "已删除"},
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty defaults to INBOX", "", "INBOX"},
		{"inbox alias lowercase", "inbox", "INBOX"},
		{"inbox alias exact", "INBOX", "INBOX"},
		{"inbox alias mixed case", "Inbox", "INBOX"},
		{"sent alias resolves via flag", "sent", "SentRaw"},
		{"sent alias mixed case", "Sent", "SentRaw"},
		{"contacts alias resolves by name", "contacts", "ContactsRaw"},
		{"display name match", "已发送", "SentRaw"},
		{"raw name match", "Drafts", "Drafts"},
		{"unknown name passes through", "Nonexistent", "Nonexistent"},
		{"surrounding whitespace trimmed", "  sent  ", "SentRaw"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveFolderName(mboxes, tc.in)
			if got != tc.want {
				t.Errorf("ResolveFolderName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// When no \Sent folder is present, the "sent" alias falls back to the literal
// "Sent" so the server can reject it with a clear error rather than failing
// resolution outright.
func TestResolveFolderNameSentFallback(t *testing.T) {
	mboxes := []Mailbox{
		{Name: "INBOX", Display: "INBOX"},
		{Name: "Drafts", Display: "草稿箱"},
	}
	if got := ResolveFolderName(mboxes, "sent"); got != "Sent" {
		t.Errorf("ResolveFolderName(sent) with no sent folder = %q, want %q", got, "Sent")
	}
}
