// Package smtpclient sends mail via SMTP.
//
// Implicit TLS (port 465) is used when SSL is enabled; otherwise the
// connection is upgraded with STARTTLS (port 587/25). Credentials are never
// sent over an unencrypted connection: if the server does not support STARTTLS
// on a non-465 port, Send returns an error instead of authenticating.
package smtpclient

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"

	"MisakaMailClient/internal/provider"
)

// plainAuth implements SASL PLAIN without the standard library's "is the
// connection TLS?" guard. It is only ever used over an already-encrypted
// connection (implicit TLS on 465, or after STARTTLS), so skipping the guard
// is safe. The standard library's smtp.PlainAuth cannot detect TLS on a
// pre-wrapped tls.Conn (port 465) and would refuse to authenticate.
type plainAuth struct {
	username, password, identity string
}

func (a *plainAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	resp := []byte(a.identity + "\x00" + a.username + "\x00" + a.password)
	return "PLAIN", resp, nil
}

func (a *plainAuth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, errors.New("unexpected server challenge")
	}
	return nil, nil
}

// dialClient establishes an SMTP client with the appropriate transport
// security for the server's port.
func dialClient(server provider.Server) (*smtp.Client, error) {
	addr := net.JoinHostPort(server.Host, strconv.Itoa(server.Port))
	host := server.Host

	if server.SSL && server.Port == 465 {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
		if err != nil {
			return nil, fmt.Errorf("connect %s: %w", addr, err)
		}
		c, err := smtp.NewClient(conn, host)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("smtp client: %w", err)
		}
		return c, nil
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("connect %s: %w", addr, err)
	}
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("smtp client: %w", err)
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		if err := c.StartTLS(&tls.Config{ServerName: host}); err != nil {
			c.Close()
			return nil, fmt.Errorf("starttls: %w", err)
		}
	} else {
		c.Close()
		return nil, fmt.Errorf("server %s does not support STARTTLS; refusing to send credentials", addr)
	}
	return c, nil
}

// Send delivers msg (raw RFC 822 bytes) from the account to all recipients.
func Send(server provider.Server, from, password string, recipients []string, msg []byte) error {
	c, err := dialClient(server)
	if err != nil {
		return err
	}
	defer c.Quit()

	auth := &plainAuth{username: from, password: password}
	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	for _, r := range recipients {
		if err := c.Rcpt(r); err != nil {
			return fmt.Errorf("rcpt %s: %w", r, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	return nil
}
