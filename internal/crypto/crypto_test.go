package crypto

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	// crypto derives the key from the keyring passphrase + the Init salt. For a
	// unit test without the keyring, drive the internals directly.
	salt := []byte("0123456789abcdef")
	key, err := scryptKey("passphrase123", salt)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}
	plain := []byte(`{"time":"2026-07-21","level":"error","message":"测试 message"}`)
	ct, err := encryptWith(key, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if ct == string(plain) {
		t.Error("ciphertext equals plaintext")
	}
	pt, err := decryptWith(key, ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(pt) != string(plain) {
		t.Errorf("round-trip mismatch: got %q want %q", pt, plain)
	}
	wrong, _ := scryptKey("different-pass", salt)
	if _, err := decryptWith(wrong, ct); err == nil {
		t.Error("expected decryption failure with wrong key")
	}
	if _, err := decryptWith(key, "not-base64!!!"); err == nil {
		t.Error("expected failure on garbage input")
	}
}
