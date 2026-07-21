package cmd

import (
	"fmt"
	"strings"

	"MisakaMailClient/internal/config"
	"MisakaMailClient/internal/credentials"
	"MisakaMailClient/internal/imapclient"
	"MisakaMailClient/internal/output"
	"MisakaMailClient/internal/provider"

	"github.com/spf13/cobra"
)

var (
	loginProvider string
	loginEmail    string
	loginName     string
	loginPassword string
	loginIMAPHost string
	loginSMTPHost string
	loginIMAPPort int
	loginSMTPPort int
	loginNoVerify bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Add a mail account (stores the password in the OS keyring)",
	Long: "Add a mail account. The password (authorization code / client-specific\n" +
		"password) is stored in the OS credential store, never on disk in plaintext.\n\n" +
		"Examples:\n" +
		"  misaka-mail login --provider qq --email me@qq.com\n" +
		"  misaka-mail login --provider aliyun-qiye --email me@corp.com\n" +
		"  misaka-mail login --email me@example.com --imap-host imap.example.com --smtp-host smtp.example.com",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve IMAP/SMTP servers from a preset and/or manual flags.
		providerName := "manual"
		var imapServer, smtpServer provider.Server
		if loginProvider != "" {
			p, ok := provider.Get(loginProvider)
			if !ok {
				return fmt.Errorf("unknown provider %q; available: %s", loginProvider, strings.Join(provider.Names(), ", "))
			}
			providerName = p.Name
			imapServer = p.IMAP
			smtpServer = p.SMTP
		}
		if loginIMAPHost != "" {
			imapServer.Host = loginIMAPHost
			providerName = "manual"
		}
		if loginIMAPPort != 0 {
			imapServer.Port = loginIMAPPort
			providerName = "manual"
		}
		if loginSMTPHost != "" {
			smtpServer.Host = loginSMTPHost
			providerName = "manual"
		}
		if loginSMTPPort != 0 {
			smtpServer.Port = loginSMTPPort
			providerName = "manual"
		}
		if imapServer.Host == "" {
			return fmt.Errorf("IMAP host is required: use --provider or --imap-host")
		}
		if smtpServer.Host == "" {
			return fmt.Errorf("SMTP host is required: use --provider or --smtp-host")
		}
		if imapServer.Port == 0 {
			imapServer.Port = 993
		}
		if smtpServer.Port == 0 {
			smtpServer.Port = 465
		}
		// Derive SSL from the port for consistency with the clients.
		imapServer.SSL = imapServer.Port == 993
		smtpServer.SSL = smtpServer.Port == 465

		// Email.
		email := loginEmail
		if email == "" {
			v, err := prompt("Email address: ")
			if err != nil {
				return err
			}
			email = v
		}
		if !strings.Contains(email, "@") {
			return fmt.Errorf("invalid email address %q", email)
		}

		// Password / authorization code.
		password := loginPassword
		if password == "" {
			v, err := promptPassword("Password / authorization code: ")
			if err != nil {
				return err
			}
			password = v
		}
		if password == "" {
			return fmt.Errorf("password is required")
		}

		// Verify the IMAP login so we fail early on a wrong auth code.
		if !loginNoVerify {
			conn, err := imapclient.Dial(imapServer, email, password)
			if err != nil {
				return fmt.Errorf("verification failed (IMAP login): %w; if the details are correct, retry with --no-verify", err)
			}
			conn.Close()
		}

		// Persist config and password.
		acc := config.Account{
			Email:       email,
			Provider:    providerName,
			DisplayName: loginName,
			IMAP:        imapServer,
			SMTP:        smtpServer,
		}
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		_, existed := cfg.Find(email)
		cfg.Add(acc)
		cfg.CurrentAccount = email
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		if err := credentials.SetPassword(email, password); err != nil {
			return fmt.Errorf("store password: %w", err)
		}

		action := "added"
		if existed {
			action = "updated"
		}
		if jsonMode {
			return output.PrintJSON(map[string]interface{}{
				"account": email,
				"action":  action,
				"current": true,
			})
		}
		verb := "Added"
		if existed {
			verb = "Updated"
		}
		fmt.Printf("%s account %s and set it as current.\n", verb, email)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVar(&loginProvider, "provider", "", fmt.Sprintf("built-in provider (%s)", strings.Join(provider.Names(), "/")))
	loginCmd.Flags().StringVar(&loginEmail, "email", "", "account email address")
	loginCmd.Flags().StringVar(&loginName, "name", "", "display name for sent mail (optional)")
	loginCmd.Flags().StringVar(&loginPassword, "password", "", "password / authorization code (if omitted, prompted securely)")
	loginCmd.Flags().StringVar(&loginIMAPHost, "imap-host", "", "IMAP server host (manual)")
	loginCmd.Flags().IntVar(&loginIMAPPort, "imap-port", 0, "IMAP server port (default 993)")
	loginCmd.Flags().StringVar(&loginSMTPHost, "smtp-host", "", "SMTP server host (manual)")
	loginCmd.Flags().IntVar(&loginSMTPPort, "smtp-port", 0, "SMTP server port (default 465)")
	loginCmd.Flags().BoolVar(&loginNoVerify, "no-verify", false, "skip IMAP login verification")
}
