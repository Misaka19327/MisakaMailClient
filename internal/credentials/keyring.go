// Package credentials stores account passwords in the OS credential store.
//
// On Windows this is the Credential Manager (wincred backend); on macOS the
// Keychain; on Linux the Secret Service / KWallet / keyctl. The password is
// never written to disk in plaintext. The backend is pinned to the OS-native
// store to avoid falling back to the passphrase-protected file backend, which
// would prompt the user for a master password.
package credentials

import (
	"fmt"
	"runtime"

	"github.com/99designs/keyring"
)

const serviceName = "misaka-mail"

func open() (keyring.Keyring, error) {
	cfg := keyring.Config{ServiceName: serviceName}
	// Pin the backend to the OS-native store.
	switch runtime.GOOS {
	case "windows":
		cfg.AllowedBackends = []keyring.BackendType{keyring.WinCredBackend}
	case "darwin":
		cfg.AllowedBackends = []keyring.BackendType{keyring.KeychainBackend}
	case "linux":
		cfg.AllowedBackends = []keyring.BackendType{keyring.SecretServiceBackend, keyring.KWalletBackend, keyring.KeyCtlBackend}
	}
	kr, err := keyring.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("open credential store: %w", err)
	}
	return kr, nil
}

// SetPassword stores the password for the given account email.
func SetPassword(email, password string) error {
	kr, err := open()
	if err != nil {
		return err
	}
	return kr.Set(keyring.Item{
		Key:   email,
		Data:  []byte(password),
		Label: "misaka-mail: " + email,
	})
}

// GetPassword retrieves the password for the given account email.
func GetPassword(email string) (string, error) {
	kr, err := open()
	if err != nil {
		return "", err
	}
	item, err := kr.Get(email)
	if err != nil {
		return "", err
	}
	return string(item.Data), nil
}

// DeletePassword removes the stored password for the given account email. A
// missing entry is not an error.
func DeletePassword(email string) error {
	kr, err := open()
	if err != nil {
		return err
	}
	if err := kr.Remove(email); err != nil && err != keyring.ErrKeyNotFound {
		return err
	}
	return nil
}
