package notification

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	Timeout  time.Duration
}

type SMTPSender struct {
	cfg SMTPConfig
}

func NewSMTPSender(cfg SMTPConfig) (*SMTPSender, error) {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Port = strings.TrimSpace(cfg.Port)
	cfg.From = strings.TrimSpace(cfg.From)
	if cfg.Host == "" || cfg.Port == "" || cfg.From == "" {
		return nil, errors.New("SMTP host, port and from address are required")
	}
	if (strings.TrimSpace(cfg.Username) == "") != (strings.TrimSpace(cfg.Password) == "") {
		return nil, errors.New("SMTP username and password must be configured together")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &SMTPSender{cfg: cfg}, nil
}

func (s *SMTPSender) Send(ctx context.Context, message Message) error {
	to := strings.TrimSpace(message.To)
	if to == "" {
		return errors.New("email recipient is required")
	}
	address := net.JoinHostPort(s.cfg.Host, s.cfg.Port)
	dialer := net.Dialer{Timeout: s.cfg.Timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("smtp dial failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client failed: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return errors.New("smtp server does not offer STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: s.cfg.Host}); err != nil {
		return fmt.Errorf("smtp STARTTLS failed: %w", err)
	}

	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp authentication failed: %w", err)
		}
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return fmt.Errorf("smtp sender rejected: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp recipient rejected: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA failed: %w", err)
	}
	body := "From: " + s.cfg.From + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + sanitizeHeader(message.Subject) + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Content-Transfer-Encoding: 8bit\r\n\r\n" + message.Text
	if _, err := writer.Write([]byte(body)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("smtp body write failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("smtp body close failed: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit failed: %w", err)
	}
	return nil
}

func sanitizeHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}
