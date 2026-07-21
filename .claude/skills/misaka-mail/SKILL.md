---
name: misaka-mail
description: Send and receive email through the misaka-mail CLI (QQ Mail / Alibaba enterprise mail). Use when the user asks to read their inbox, view a specific message, send or reply to emails (including HTML and file attachments), or manage mail accounts from the terminal. Outputs JSON when asked to parse results.
---

# misaka-mail

`misaka-mail` is a third-party email CLI installed on this system. It supports
multiple accounts, inbox reading, sending/replying (with attachments and HTML),
and JSON output. Credentials are stored in the OS credential store
(Windows Credential Manager), never as plaintext on disk.

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
