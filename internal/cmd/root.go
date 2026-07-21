// Package cmd implements the misaka-mail CLI commands.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"MisakaMailClient/internal/config"
	"MisakaMailClient/internal/credentials"
	"MisakaMailClient/internal/logging"
	"MisakaMailClient/internal/output"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	jsonMode        bool
	overrideAccount string
	appVersion      string
)

var rootCmd = &cobra.Command{
	Use:   "misaka-mail",
	Short: "A third-party email CLI for sending and receiving mail",
	Long: "MisakaMail - 收发邮件的命令行工具。\n\n" +
		"支持多账号管理、收件箱读取、发送/回复邮件(含附件与 HTML)、JSON 输出。\n" +
		"凭据保存在系统密钥库(Windows 凭据管理器),非明文。",
}

// lastCommand records the command path being executed, for error logging.
var lastCommand string

// Execute runs the root command. version is reported via --version.
func Execute(version string) {
	appVersion = version
	rootCmd.Version = version
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		lastCommand = cmd.CommandPath()
		// Don't log the log commands themselves (avoids self-referential noise).
		if !strings.HasPrefix(cmd.CommandPath(), "misaka-mail log") {
			acct := ""
			if acc, err := resolveAccount(); err == nil && acc != nil {
				acct = acc.Email
			}
			logging.Write(logging.LevelInfo, cmd.CommandPath(), acct, "started")
		}
		return nil
	}
	if cfg, err := config.Load(); err == nil {
		logging.Init(cfg.LoggingConfig())
	}
	if err := rootCmd.Execute(); err != nil {
		logging.Write(logging.LevelError, lastCommand, "", err.Error())
		output.PrintError(jsonMode, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "output results as JSON")
	rootCmd.PersistentFlags().StringVar(&overrideAccount, "account", "", "use this account for the command (overrides the current account)")
}

// stdinReader is the shared buffered reader for interactive prompts.
var stdinReader = bufio.NewReader(os.Stdin)

// resolveAccount returns the account to use for this command, honoring the
// --account override.
func resolveAccount() (*config.Account, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if overrideAccount != "" {
		a, ok := cfg.Find(overrideAccount)
		if !ok {
			return nil, fmt.Errorf("account %q not found; run 'misaka-mail login' first", overrideAccount)
		}
		return a, nil
	}
	a, ok := cfg.Current()
	if !ok {
		return nil, fmt.Errorf("no account selected; run 'misaka-mail login' or 'misaka-mail use <email>'")
	}
	return a, nil
}

// accountPassword returns the stored password for the account.
func accountPassword(a *config.Account) (string, error) {
	p, err := credentials.GetPassword(a.Email)
	if err != nil {
		return "", fmt.Errorf("no stored password for %s (run 'misaka-mail login'): %w", a.Email, err)
	}
	return p, nil
}

// prompt reads a single trimmed line from stdin after printing question to
// stderr (so JSON stdout stays clean).
func prompt(question string) (string, error) {
	fmt.Fprint(os.Stderr, question)
	line, err := stdinReader.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// promptPassword reads a password without echo when stdin is a terminal; when
// stdin is piped it reads a normal line (so logins can be scripted).
func promptPassword(question string) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, question)
		pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		return string(pwd), err
	}
	return prompt(question)
}

// parseSeq parses a positive sequence number argument.
func parseSeq(s string) (uint32, error) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid sequence number %q", s)
	}
	if n == 0 {
		return 0, fmt.Errorf("sequence number must be >= 1")
	}
	return uint32(n), nil
}
