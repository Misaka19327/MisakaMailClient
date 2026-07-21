package credentials

import "testing"

// TestRoundTrip stores and retrieves a password through the OS credential
// store, then removes it. This exercises the real keyring backend on the host.
func TestRoundTrip(t *testing.T) {
	const email = "misaka-mail-test@example.com"
	const secret = "super-secret-authorization-code-123"

	t.Cleanup(func() { _ = DeletePassword(email) })

	if err := SetPassword(email, secret); err != nil {
		t.Fatalf("SetPassword: %v", err)
	}
	got, err := GetPassword(email)
	if err != nil {
		t.Fatalf("GetPassword: %v", err)
	}
	if got != secret {
		t.Errorf("GetPassword: got %q want %q", got, secret)
	}
	if err := DeletePassword(email); err != nil {
		t.Fatalf("DeletePassword: %v", err)
	}
	if _, err := GetPassword(email); err == nil {
		t.Error("GetPassword after delete: expected error, got nil")
	}
}
