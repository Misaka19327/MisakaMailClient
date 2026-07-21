// Package crypto provides shared AES-256-GCM encryption for sensitive local
// data (logs and contact caches). The key is derived (via scrypt) from a
// passphrase stored in the OS keyring; the KDF salt is held by the caller
// (config) and injected via Init. One passphrase (set via 'misaka-mail log
// key') encrypts all local data.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"MisakaMailClient/internal/credentials"

	"golang.org/x/crypto/scrypt"
)

// KeyID is the keyring entry holding the encryption passphrase. Named "log-key"
// for backward compatibility; it now encrypts all local data.
const KeyID = "log-key"

var salt []byte

// Init sets the KDF salt (base64). Must be called before Encrypt/Decrypt/SetKey.
func Init(saltB64 string) {
	if s, err := base64.StdEncoding.DecodeString(saltB64); err == nil {
		salt = s
	}
}

// HasKey reports whether an encryption passphrase is stored.
func HasKey() bool {
	_, err := credentials.GetPassword(KeyID)
	return err == nil
}

func deriveKey(pass string) ([]byte, error) {
	if len(salt) == 0 {
		return nil, fmt.Errorf("no salt configured")
	}
	return scryptKey(pass, salt)
}

// scryptKey derives a 32-byte AES key from a passphrase and salt.
func scryptKey(pass string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(pass), salt, 1<<15, 8, 1, 32)
}

func passphrase() (string, error) {
	p, err := credentials.GetPassword(KeyID)
	if err != nil {
		return "", fmt.Errorf("no encryption key set; run 'misaka-mail log key'")
	}
	return p, nil
}

// encryptWith encrypts plain with AES-256-GCM using key, returning
// base64(nonce || ciphertext+tag).
func encryptWith(key, plain []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nil, nonce, plain, nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ct...)), nil
}

// decryptWith decrypts a base64(nonce || ciphertext+tag) value using key.
func decryptWith(key []byte, b64 string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(raw) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}
	return gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
}

// Encrypt encrypts plaintext with AES-256-GCM and returns
// base64(nonce || ciphertext+tag).
func Encrypt(plain []byte) (string, error) {
	pass, err := passphrase()
	if err != nil {
		return "", err
	}
	key, err := deriveKey(pass)
	if err != nil {
		return "", err
	}
	return encryptWith(key, plain)
}

// Decrypt decrypts a base64(nonce || ciphertext+tag) value produced by Encrypt.
func Decrypt(b64 string) ([]byte, error) {
	pass, err := passphrase()
	if err != nil {
		return nil, err
	}
	key, err := deriveKey(pass)
	if err != nil {
		return nil, err
	}
	return decryptWith(key, b64)
}

// SetKey stores the passphrase (min 6 characters). It generates a salt on first
// set and preserves it afterwards. It returns the salt (base64) to persist and
// whether an existing key was replaced (in which case data encrypted with the
// old key is no longer decryptable).
func SetKey(pass string) (saltB64 string, replaced bool, err error) {
	if len(pass) < 6 {
		return "", false, fmt.Errorf("key must be at least 6 characters")
	}
	existing, err := credentials.GetPassword(KeyID)
	replaced = err == nil && existing != ""
	if len(salt) == 0 {
		s := make([]byte, 16)
		if _, err := rand.Read(s); err != nil {
			return "", false, err
		}
		salt = s
	}
	if err := credentials.SetPassword(KeyID, pass); err != nil {
		return "", false, err
	}
	return base64.StdEncoding.EncodeToString(salt), replaced, nil
}

// PurgeKey removes the stored passphrase.
func PurgeKey() error {
	return credentials.DeletePassword(KeyID)
}
