package contacts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"MisakaMailClient/internal/config"
	"MisakaMailClient/internal/crypto"
)

// Cache is the encrypted, per-account stored contact list.
type Cache struct {
	Account  string    `json:"account"`
	PulledAt time.Time `json:"pulled_at"`
	Contacts []Contact `json:"contacts"`
}

func storePath(account string) (string, error) {
	d, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "contacts-"+account+".enc"), nil
}

// Save encrypts and writes the contact list for the account.
func Save(account string, list []Contact) error {
	data, err := json.Marshal(Cache{
		Account:  account,
		PulledAt: time.Now(),
		Contacts: list,
	})
	if err != nil {
		return err
	}
	enc, err := crypto.Encrypt(data)
	if err != nil {
		return err
	}
	p, err := storePath(account)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	return os.WriteFile(p, []byte(enc), 0o600)
}

// Load reads and decrypts the contact cache for the account. It returns
// (nil, nil) when no cache exists.
func Load(account string) (*Cache, error) {
	p, err := storePath(account)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	dec, err := crypto.Decrypt(string(data))
	if err != nil {
		return nil, fmt.Errorf("decrypt contacts cache: %w (run 'misaka-mail log key' with the matching key)", err)
	}
	var c Cache
	if err := json.Unmarshal(dec, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// PurgeAll removes every contacts-*.enc cache file (used when the encryption
// key changes, leaving old caches undecryptable).
func PurgeAll() error {
	d, err := config.Dir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(d)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "contacts-") && strings.HasSuffix(e.Name(), ".enc") {
			_ = os.Remove(filepath.Join(d, e.Name()))
		}
	}
	return nil
}

func sourceRank(s string) int {
	switch s {
	case "vcard":
		return 3
	case "sent":
		return 2
	case "inbox":
		return 1
	}
	return 0
}

// Merge combines two contact lists, deduplicating by email (lowercase) and
// preferring non-empty names; source takes the higher-ranked value
// (vcard > sent > inbox).
func Merge(existing, fresh []Contact) []Contact {
	byEmail := make(map[string]*Contact)
	add := func(c Contact) {
		email := strings.ToLower(strings.TrimSpace(c.Email))
		if email == "" {
			return
		}
		if ex, ok := byEmail[email]; ok {
			if ex.Name == "" && c.Name != "" {
				ex.Name = c.Name
			}
			if sourceRank(c.Source) > sourceRank(ex.Source) {
				ex.Source = c.Source
			}
			return
		}
		cc := c
		byEmail[email] = &cc
	}
	for _, c := range existing {
		add(c)
	}
	for _, c := range fresh {
		add(c)
	}
	out := make([]Contact, 0, len(byEmail))
	for _, c := range byEmail {
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Email < out[j].Email
	})
	return out
}
