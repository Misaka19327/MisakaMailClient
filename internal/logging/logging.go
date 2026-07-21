// Package logging writes encrypted, JSON-formatted log entries to disk.
//
// Each entry is serialized to JSON, encrypted with AES-256-GCM (key derived
// from a user passphrase via scrypt), and appended as one base64 line to a
// per-day file under <configdir>/misaka-mail/logs/YYYY-MM-DD.enc. The
// passphrase is stored in the OS keyring; the KDF salt is stored in config
// (non-secret). Without a passphrase set, logging is a silent no-op so the CLI
// keeps working.
package logging

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"MisakaMailClient/internal/config"
	"MisakaMailClient/internal/credentials"

	"golang.org/x/crypto/scrypt"
)

// Level is a log severity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// ParseLevel parses a level string.
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level %q (debug|info|warn|error)", s)
	}
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

const logKeyID = "log-key"

// Entry is one log record.
type Entry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Command string    `json:"command,omitempty"`
	Account string    `json:"account,omitempty"`
	Message string    `json:"message"`
}

var (
	cfg       config.Logging
	level     = LevelError
	retention = 7
)

// Init applies the logging configuration and purges expired log files.
func Init(c config.Logging) {
	cfg = c
	if l, err := ParseLevel(c.Level); err == nil && c.Level != "" {
		level = l
	} else {
		level = LevelError
	}
	if c.RetentionDays > 0 {
		retention = c.RetentionDays
	} else {
		retention = 7
	}
	purgeOldLogs(retention)
}

// Salt returns the configured KDF salt (empty if none).
func Salt() string { return cfg.Salt }

func logsDir() (string, error) {
	d, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "logs"), nil
}

func logFileForToday() string {
	d, _ := logsDir()
	return filepath.Join(d, time.Now().Format("2006-01-02")+".enc")
}

func deriveKey(pass string, salt []byte) ([]byte, error) {
	return scrypt.Key([]byte(pass), salt, 1<<15, 8, 1, 32)
}

func getPassphrase() string {
	p, err := credentials.GetPassword(logKeyID)
	if err != nil {
		return ""
	}
	return p
}

func encrypt(key, plaintext []byte) (string, error) {
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
	ct := gcm.Seal(nil, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(append(nonce, ct...)), nil
}

func decrypt(key []byte, b64 string) ([]byte, error) {
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

// Write appends a log entry if the level is enabled and a key is set. It is
// best-effort and never returns an error that callers must handle.
func Write(lvl Level, command, account, message string) {
	if lvl < level {
		return
	}
	pass := getPassphrase()
	if pass == "" {
		return
	}
	salt, err := base64.StdEncoding.DecodeString(cfg.Salt)
	if err != nil || len(salt) == 0 {
		return
	}
	key, err := deriveKey(pass, salt)
	if err != nil {
		return
	}
	data, err := json.Marshal(Entry{
		Time:    time.Now(),
		Level:   lvl.String(),
		Command: command,
		Account: account,
		Message: message,
	})
	if err != nil {
		return
	}
	line, err := encrypt(key, data)
	if err != nil {
		return
	}
	dir, err := logsDir()
	if err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(logFileForToday(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}

// Read returns decrypted entries at or above minLevel, optionally since a time.
func Read(minLevel Level, since time.Time) ([]Entry, error) {
	pass := getPassphrase()
	if pass == "" {
		return nil, fmt.Errorf("no log encryption key set; run 'misaka-mail log key'")
	}
	salt, err := base64.StdEncoding.DecodeString(cfg.Salt)
	if err != nil || len(salt) == 0 {
		return nil, fmt.Errorf("no log salt configured; run 'misaka-mail log key'")
	}
	key, err := deriveKey(pass, salt)
	if err != nil {
		return nil, err
	}
	dir, err := logsDir()
	if err != nil {
		return nil, err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, err
	}
	out := []Entry{}
	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, fi.Name()))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			dec, err := decrypt(key, line)
			if err != nil {
				continue // wrong key / corrupt: skip
			}
			var e Entry
			if json.Unmarshal(dec, &e) != nil {
				continue
			}
			if lv, err := ParseLevel(e.Level); err == nil && lv < minLevel {
				continue
			}
			if !since.IsZero() && e.Time.Before(since) {
				continue
			}
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out, nil
}

// SetKey stores the passphrase. On first set it generates a salt; on later
// sets the salt is preserved. It returns the salt to persist and whether an
// existing key was replaced (old logs become undecryptable).
func SetKey(pass string) (salt string, replaced bool, err error) {
	if len(pass) < 6 {
		return "", false, fmt.Errorf("key must be at least 6 characters")
	}
	existing, err := credentials.GetPassword(logKeyID)
	replaced = err == nil && existing != ""
	salt = cfg.Salt
	if salt == "" {
		s := make([]byte, 16)
		if _, err := rand.Read(s); err != nil {
			return "", false, err
		}
		salt = base64.StdEncoding.EncodeToString(s)
	}
	if err := credentials.SetPassword(logKeyID, pass); err != nil {
		return "", false, err
	}
	return salt, replaced, nil
}

func purgeOldLogs(days int) {
	dir, err := logsDir()
	if err != nil {
		return
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	for _, fi := range files {
		if fi.IsDir() {
			continue
		}
		t, err := time.ParseInLocation("2006-01-02", strings.TrimSuffix(fi.Name(), ".enc"), time.Local)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, fi.Name()))
		}
	}
}

// PurgeAll removes all log files.
func PurgeAll() error {
	dir, err := logsDir()
	if err != nil {
		return err
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, fi := range files {
		if !fi.IsDir() {
			_ = os.Remove(filepath.Join(dir, fi.Name()))
		}
	}
	return nil
}
