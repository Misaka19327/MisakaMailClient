package logging

import "testing"

func TestParseLevel(t *testing.T) {
	cases := map[string]Level{
		"debug":   LevelDebug,
		"INFO":    LevelInfo,
		"warn":    LevelWarn,
		"warning": LevelWarn,
		"error":   LevelError,
	}
	for s, want := range cases {
		got, err := ParseLevel(s)
		if err != nil || got != want {
			t.Errorf("ParseLevel(%q) = %v, %v; want %v", s, got, err, want)
		}
	}
	if _, err := ParseLevel("nope"); err == nil {
		t.Error("ParseLevel(invalid) expected error")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	salt := []byte("0123456789abcdef")
	key, err := deriveKey("passphrase123", salt)
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}
	plaintext := []byte(`{"time":"2026-07-21","level":"error","message":"测试 message"}`)
	ct, err := encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if ct == string(plaintext) {
		t.Error("ciphertext equals plaintext")
	}
	pt, err := decrypt(key, ct)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(pt) != string(plaintext) {
		t.Errorf("round-trip mismatch: got %q want %q", pt, plaintext)
	}
	// A wrong key must fail to decrypt.
	wrong, _ := deriveKey("different-pass", salt)
	if _, err := decrypt(wrong, ct); err == nil {
		t.Error("expected decryption failure with wrong key")
	}
	// Garbage input fails gracefully.
	if _, err := decrypt(key, "not-base64!!!"); err == nil {
		t.Error("expected failure on garbage input")
	}
}
