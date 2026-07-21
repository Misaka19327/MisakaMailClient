// Package config manages the on-disk account configuration.
//
// Only non-sensitive metadata (email, provider, server addresses, display
// name) is stored on disk. Passwords live in the OS credential store via the
// credentials package and are never written to this file.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"MisakaMailClient/internal/provider"
)

// Account is a configured mail account (without its password).
type Account struct {
	Email       string          `json:"email"`
	Provider    string          `json:"provider"`
	DisplayName string          `json:"display_name,omitempty"`
	IMAP        provider.Server `json:"imap"`
	SMTP        provider.Server `json:"smtp"`
}

// Config is the persisted application configuration.
type Config struct {
	CurrentAccount string    `json:"current_account,omitempty"`
	Accounts       []Account `json:"accounts"`
}

// Dir returns the configuration directory path.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "misaka-mail"), nil
}

func path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
}

// Load reads the configuration from disk. A missing file yields an empty
// configuration rather than an error.
func Load() (*Config, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Save writes the configuration to disk with restrictive permissions.
func (c *Config) Save() error {
	d, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0o700); err != nil {
		return err
	}
	p, err := path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// Find returns the account with the given email, if present.
func (c *Config) Find(email string) (*Account, bool) {
	for i := range c.Accounts {
		if c.Accounts[i].Email == email {
			return &c.Accounts[i], true
		}
	}
	return nil, false
}

// Add upserts an account. If an account with the same email exists it is
// replaced.
func (c *Config) Add(a Account) {
	for i := range c.Accounts {
		if c.Accounts[i].Email == a.Email {
			c.Accounts[i] = a
			return
		}
	}
	c.Accounts = append(c.Accounts, a)
}

// Remove deletes the account with the given email. It reports whether an
// account was removed and clears the current selection if it pointed at it.
func (c *Config) Remove(email string) bool {
	for i := range c.Accounts {
		if c.Accounts[i].Email == email {
			c.Accounts = append(c.Accounts[:i], c.Accounts[i+1:]...)
			if c.CurrentAccount == email {
				c.CurrentAccount = ""
			}
			return true
		}
	}
	return false
}

// Current returns the active account. If no account is explicitly selected but
// exactly one account is configured, that account is used.
func (c *Config) Current() (*Account, bool) {
	if c.CurrentAccount != "" {
		return c.Find(c.CurrentAccount)
	}
	if len(c.Accounts) == 1 {
		return &c.Accounts[0], true
	}
	return nil, false
}
