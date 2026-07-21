---
name: misaka-mail
description: Send and receive email through the misaka-mail CLI (QQ Mail / Alibaba enterprise mail). Use when the user asks to read their inbox, view a specific message, send or reply to emails (including HTML and file attachments), or manage mail accounts from the terminal. Outputs JSON when asked to parse results.
---

# misaka-mail

`misaka-mail` is a third-party email CLI installed on this system. It supports
multiple accounts, inbox reading, sending/replying (with attachments and HTML),
and JSON output. Credentials are stored in the OS credential store
(Windows Credential Manager), never as plaintext on disk.

## Installation

Pick one, then run `misaka-mail install` to add it to PATH:

- **From a release (no Go needed)**: download
  `misaka-mail_windows_amd64.tar.gz` (or the matching archive) from
  https://github.com/Misaka19327/MisakaMailClient/releases, extract
  `misaka-mail`/`misaka-mail.exe`, then run `misaka-mail install` from that
  directory.
- **From source (Go 1.26+)**:
  ```bash
  git clone https://github.com/Misaka19327/MisakaMailClient
  cd MisakaMailClient
  go env -w GOPROXY=https://goproxy.cn,direct   # China: default proxy is blocked
  go install ./cmd/misaka-mail
  misaka-mail install                            # add the binary dir to PATH
  ```

`misaka-mail install` adds the binary's directory to the user PATH (Windows:
writes `HKCU\Environment\Path` and broadcasts `WM_SETTINGCHANGE`; Unix: appends
an `export PATH=...` to `~/.bashrc`, `~/.zshrc`, `~/.profile`). Open a new
terminal afterwards. Upgrade with `misaka-mail update` (checks GitHub releases,
verifies SHA256 against `checksums.txt`, atomically replaces the binary).

## When to use

Use this skill whenever the user wants to work with email from the command line:
list/read the inbox, send or reply to messages, attach files, or add/switch
mail accounts. Prefer it over writing ad-hoc IMAP/SMTP scripts.

## Prerequisites

- The binary is installed at `C:\Users\Misaka19327\go\bin\misaka-mail.exe`.
  If `misaka-mail` is not on PATH, run `misaka-mail install` to add its
  directory to the user PATH automatically (or call it by full path).
- At least one account must be configured via `misaka-mail login` first.
- Login uses an **authorization code / client-specific password**, NOT the
  account's main web-login password:
  - QQ 邮箱: enable IMAP/SMTP in QQ Mail settings and generate an 授权码.
  - 阿里企业邮箱: the admin must enable third-party client access; generate a
    客户端专用密码 (third-party client security password).

## Output conventions

- Pass `--json` to get structured output for programmatic parsing. Always use
  `--json` when you need to read fields reliably.
- Errors go to stderr as `{"error": "..."}` when `--json` is set, with a
  non-zero exit code. Check exit codes.
- The "current account" is used by default; override per-command with
  `--account <email>`.

## Commands

### Install

```bash
# Add the executable's directory to the user PATH (Windows: HKCU\Environment;
# Unix: ~/.bashrc, ~/.zshrc, ~/.profile). Idempotent.
misaka-mail install --json
misaka-mail install --dir /custom/path   # add a specific directory
```
Run once after `go install` or after placing a release binary, then open a new
terminal. JSON: `{"directory": "...", "added": bool}`.

### Account management

```bash
# Add an account (prompts for the auth code securely if --password omitted).
misaka-mail login --provider qq --email me@qq.com
misaka-mail login --provider aliyun-qiye --email me@corp.com
# Manual servers:
misaka-mail login --email me@example.com --imap-host imap.example.com --smtp-host smtp.example.com

misaka-mail accounts            # list accounts (* = current)
misaka-mail use <email>         # switch current account
misaka-mail whoami              # show current account + servers
misaka-mail whoami --json
misaka-mail logout <email>      # remove account + delete stored password
```

Providers: `qq`, `aliyun-qiye`, `aliyun-qiye-hk`.

### Reading mail

```bash
# List inbox (newest first). <seq> is the sequence number used by read/reply.
misaka-mail inbox --limit 20 --json
misaka-mail inbox --unread          # only unread

# Read a message; --save-attachments writes attachments to a directory.
misaka-mail read 5 --json
misaka-mail read 5 --save-attachments ./downloads
```

Inbox JSON: `{"account": "...", "messages": [{seq, uid, subject, from,
from_name, date, seen, has_attachments}, ...]}`.

Read JSON: `{"account": "...", "message": {message_id, subject, from, from_name,
to, cc, date, text_body, html_body, attachments: [{filename, mime_type, size}]},
"saved_attachments": [...]}` (saved_attachments only with --save-attachments).

### Sending mail

```bash
# Plain text, multiple recipients (repeat --to or comma-separate).
misaka-mail send --to a@x.com --to b@y.com --subject "Hi" --body "Hello"

# HTML body from a file + attachments.
misaka-mail send --to a@x.com --subject "Report" \
  --html-file ./report.html --attach ./data.csv --attach ./chart.png

# CC/BCC (Bcc is envelope-only; recipients do not see each other).
misaka-mail send --to a@x.com --cc b@y.com --bcc c@z.com --subject "X" --body "Y"
```

Send JSON: `{"sent": true, "account": "...", "message_id": "...",
"recipients": [...]}`.

Body flags: `--body` / `--body-file` (plain text), `--html` / `--html-file`
(HTML). At least one body and one recipient are required.

### Replying

```bash
# Reply to message <seq>; sets In-Reply-To, References, and "Re:" subject.
misaka-mail reply 5 --body "Thanks, got it."
misaka-mail reply 5 --all --html-file ./reply.html --attach ./signed.pdf
```

Reply JSON: `{"sent": true, "account": "...", "message_id": "...",
"in_reply_to": "...", "recipients": [...]}`.

### Self-update

```bash
# Check whether a newer release exists on GitHub (does not apply).
misaka-mail update --check --json

# Apply the latest release (atomically replaces the running binary).
misaka-mail update

# Pin to a specific version.
misaka-mail update --version 0.2.0

# Override the GitHub repo for this invocation.
misaka-mail update --repo owner/repo --check
```

The GitHub repo is baked in at build time via `-ldflags`; if unset, pass
`--repo owner/repo`. Downloads are SHA256-verified against the release's
`checksums.txt` (skip with `--no-verify`). Set `GITHUB_TOKEN` for higher API
rate limits or private repos. Update JSON: `{"updated": true, "version": "..."}`
or check `{"current": "...", "latest": "...", "update_available": bool}`.

## Logging

Encrypted, JSON logs. Set a key (min 6 chars) first; without it, logging is off
and commands still work normally.

```bash
misaka-mail log key               # set/change the encryption key (prompted, min 6)
misaka-mail log level info        # debug|info|warn|error (default error)
misaka-mail log retention 14      # days to keep (default 7)
misaka-mail log show              # decrypt + print (JSON by default)
misaka-mail log show --level error --since 48h --text
misaka-mail log purge             # delete all log files
```

Each entry is AES-256-GCM encrypted (key derived via scrypt from the passphrase,
stored in the OS keyring) and appended as one base64 line to
`%APPDATA%\misaka-mail\logs\YYYY-MM-DD.enc`. Commands log an `info` entry on
start (when level allows) and an `error` entry on failure. `log show` outputs
JSON by default; `--text` for human-readable. Entry JSON: `{time, level,
command, account, message}`.

## Contacts

```bash
misaka-mail contacts                  # default: read local encrypted cache (fast)
misaka-mail contacts --refresh        # re-pull from server, replace cache
misaka-mail contacts --merge          # re-pull and merge with cache
misaka-mail contacts --include-inbox --limit 500
```

Tries a Contacts folder (vCard) first, then scans the Sent folder for
recipients (portable fallback — IMAP has no standard address-book API).
`--include-inbox` also collects INBOX senders. Output includes a `notes` array
describing which folders were found. Each contact has `name`, `email`, `source`
(`vcard`|`sent`|`inbox`).

Results are cached locally per account, encrypted with the app encryption key
(set via `misaka-mail log key`); by default the cache is returned, `--refresh`
replaces it, `--merge` combines. Without a key, contacts are pulled but not
cached. JSON includes `source` (`cache`|`refresh`|`merge`|`server`), `count`,
`pulled_at`, `contacts`, and `notes`.

## Typical workflow

1. `misaka-mail login --provider <p> --email <e>` (once per account).
2. `misaka-mail inbox --json` → pick a `seq`.
3. `misaka-mail read <seq> --json` → inspect body/attachments.
4. `misaka-mail reply <seq> --body "..."` or
   `misaka-mail send --to ... --subject ... --body ... --attach <path>`.

## Notes / gotchas

- Sequence numbers (`seq`) come from `inbox` and are valid for the current
  mailbox state; read/reply immediately after listing.
- `read` uses BODY.PEEK — it does NOT mark messages as seen.
- SMTP is always encrypted: implicit TLS on port 465, or STARTTLS otherwise.
  The CLI refuses to send credentials over an unencrypted connection.
- If login verification fails but the details are correct, retry with
  `--no-verify`.
