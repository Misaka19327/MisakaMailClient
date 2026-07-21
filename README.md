# MisakaMail

A third-party email CLI for sending and receiving mail from the terminal.
Built in Go; installs as a single binary. Supports multiple accounts, inbox
reading, sending/replying (including HTML and file attachments), and JSON
output. Credentials are stored in the OS credential store (Windows Credential
Manager), never as plaintext on disk.

Built-in presets: **QQ 邮箱** and **阿里企业邮箱** (Alibaba Cloud Mail), plus
manual IMAP/SMTP configuration for any provider.

## Install

Requirements: Go 1.26+.

```powershell
# Use a reachable Go proxy (default proxy.golang.org is often unreachable in China).
go env -w GOPROXY=https://goproxy.cn,direct

# From the project root:
go install ./cmd/misaka-mail
```

To enable `misaka-mail update` without passing `--repo` each time, bake the
GitHub repository in at build time:

```powershell
go install -ldflags "-X MisakaMailClient/internal/updater.defaultOwner=OWNER -X MisakaMailClient/internal/updater.defaultRepo=REPO" ./cmd/misaka-mail
```

The binary is placed at `%USERPROFILE%\go\bin\misaka-mail.exe`. Add it to PATH
(if not already):

```powershell
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";$env:USERPROFILE\go\bin", "User")
```

Then open a new terminal and verify:

```powershell
misaka-mail --version
```

## Getting an authorization code

Login uses an **authorization code / client-specific password**, not the
account's main web-login password.

- **QQ 邮箱**: QQ 邮箱 → 设置 → 账户 → 开启 IMAP/SMTP 服务 → 生成授权码.
- **阿里企业邮箱**: 管理员开启第三方客户端登录权限;账户设置中生成"第三方客户端安全密码".
  - Servers (built-in): IMAP `imap.qiye.aliyun.com:993` SSL,
    SMTP `smtp.qiye.aliyun.com:465` SSL. (HK region: `imaphk`/`smtphk`.)

## Usage

### Account management

```bash
misaka-mail login --provider qq --email me@qq.com
misaka-mail login --provider aliyun-qiye --email me@corp.com
misaka-mail login --email me@example.com --imap-host imap.example.com --smtp-host smtp.example.com

misaka-mail accounts            # list accounts (* = current)
misaka-mail use <email>         # switch current account
misaka-mail whoami              # show current account and servers
misaka-mail logout <email>      # remove account and delete stored password
```

Global flags: `--json` (structured output), `--account <email>` (override the
current account for one command).

### Reading mail

```bash
misaka-mail inbox --limit 20            # list inbox, newest first
misaka-mail inbox --unread --json       # only unread, JSON output
misaka-mail read 5 --json               # read message <seq>
misaka-mail read 5 --save-attachments ./downloads
```

### Sending mail

```bash
misaka-mail send --to a@x.com --to b@y.com --subject "Hi" --body "Hello"
misaka-mail send --to a@x.com --subject "Report" \
  --html-file ./report.html --attach ./data.csv --attach ./chart.png
misaka-mail send --to a@x.com --cc b@y.com --bcc c@z.com --subject "X" --body "Y"
```

Body: `--body` / `--body-file` (plain text), `--html` / `--html-file` (HTML).
Attachments: `--attach <path>` (repeatable). At least one body and one recipient
are required. Bcc is envelope-only — recipients cannot see each other.

### Replying

```bash
misaka-mail reply 5 --body "Thanks, got it."
misaka-mail reply 5 --all --html-file ./reply.html --attach ./signed.pdf
```

Reply sets `In-Reply-To`, `References`, and a `Re:` subject automatically.
`--all` replies to all recipients.

## JSON output

Every data command supports `--json`. Errors are emitted to stderr as
`{"error": "..."}` with a non-zero exit code. See the skill at
`.claude/skills/misaka-mail/SKILL.md` for field shapes.

## Self-update

`misaka-mail` can update itself from GitHub Releases:

```bash
misaka-mail update --check          # check only, do not apply
misaka-mail update                  # update to the latest release
misaka-mail update --version 0.2.0  # pin a specific version
misaka-mail update --repo owner/repo --check   # override the repo for one call
misaka-mail update --no-verify      # skip SHA256 checksum verification (insecure)
```

The running binary is replaced atomically (on Windows the in-use `.exe` is
renamed first). Downloads are SHA256-verified against the release's
`checksums.txt`. Set `GITHUB_TOKEN` for higher GitHub API rate limits or to
update from a private repository.

### Publishing releases

The repository to check is set at build time via `-ldflags` (see Install) or per
invocation via `--repo`. To produce releases the updater can install, use the
`go-selfupdate` tool, which packages binaries and generates `checksums.txt`:

```powershell
go install github.com/creativeprojects/go-selfupdate/cmd/go-selfupdate@latest
# Build the binary for the current platform, then package + (optionally) upload:
go-selfupdate -o release/ 0.2.0
# Or automate on tag push with the creativeprojects/go-selfupdate GitHub Action.
```

Asset naming follows go-selfupdate's convention
(`misaka-mail_<version>_<os>_<arch>.zip` / `.tar.gz`), which the updater matches
against the current `GOOS`/`GOARCH` automatically.

## How credentials are stored

- **Password / authorization code**: OS credential store
  (Windows Credential Manager via the `wincred` backend). Never written to disk
  in plaintext.
- **Account metadata** (email, provider, servers): `%APPDATA%\misaka-mail\config.json`,
  with no password fields.

## Project layout

```
cmd/misaka-mail/        binary entry point
internal/
  cmd/                  cobra commands (login, inbox, read, send, reply, ...)
  config/               account config persistence (no passwords)
  credentials/          OS keyring access
  provider/             built-in server presets
  imapclient/           IMAP receive (go-imap)
  smtpclient/           SMTP send (implicit TLS / STARTTLS)
  message/              MIME build + parse (go-message)
  output/               JSON / text rendering
  updater/              GitHub-Releases self-update (go-selfupdate)
.claude/skills/misaka-mail/SKILL.md   assistant skill
```

## Development

```bash
go mod tidy
go build ./...
go test ./...
go vet ./...
```
