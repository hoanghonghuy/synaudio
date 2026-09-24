package config

import (
	"errors"
	"os"
	"strings"
	"time"
)

const (
	EmailModeDisabled = "disabled"
	EmailModeSMTP     = "smtp"
	EmailModeTerminal = "terminal"
	EmailModeConsole  = "console"
)

type EmailConfig struct {
	Mode          string
	AppPublicURL  string
	PayloadSecret string
	SMTPHost      string
	SMTPPort      string
	SMTPUsername  string
	SMTPPassword  string
	SMTPFrom      string
	SMTPTimeout   time.Duration
}

func LoadEmail(appEnv, appPublicURL string) (EmailConfig, error) {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("EMAIL_MODE")))
	if mode == "" {
		mode = EmailModeDisabled
	}
	cfg := EmailConfig{
		Mode:          mode,
		AppPublicURL:  strings.TrimRight(strings.TrimSpace(appPublicURL), "/"),
		PayloadSecret: strings.TrimSpace(os.Getenv("EMAIL_PAYLOAD_SECRET")),
		SMTPHost:      strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:      strings.TrimSpace(os.Getenv("SMTP_PORT")),
		SMTPUsername:  strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:  strings.TrimSpace(os.Getenv("SMTP_PASSWORD")),
		SMTPFrom:      strings.TrimSpace(os.Getenv("SMTP_FROM")),
		SMTPTimeout:   10 * time.Second,
	}
	if cfg.SMTPPort == "" {
		cfg.SMTPPort = "587"
	}
	if raw := strings.TrimSpace(os.Getenv("SMTP_TIMEOUT")); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			return EmailConfig{}, errors.New("SMTP_TIMEOUT must be a positive duration")
		}
		cfg.SMTPTimeout = value
	}
	if cfg.Mode != EmailModeDisabled && cfg.Mode != EmailModeSMTP && cfg.Mode != EmailModeTerminal && cfg.Mode != EmailModeConsole {
		return EmailConfig{}, errors.New("EMAIL_MODE must be disabled, smtp, or terminal")
	}
	if strings.EqualFold(appEnv, EnvProduction) && cfg.Mode != EmailModeSMTP {
		return EmailConfig{}, errors.New("EMAIL_MODE=smtp is required in production")
	}
	if cfg.Mode == EmailModeDisabled {
		return cfg, nil
	}
	if cfg.AppPublicURL == "" {
		if !strings.EqualFold(appEnv, EnvProduction) {
			cfg.AppPublicURL = "http://localhost:5173"
		} else {
			return EmailConfig{}, errors.New("APP_PUBLIC_URL is required when transactional email is enabled")
		}
	}
	if len(cfg.PayloadSecret) < 32 {
		if !strings.EqualFold(appEnv, EnvProduction) {
			cfg.PayloadSecret = "development-only-email-payload-secret-at-least-32-bytes"
		} else {
			return EmailConfig{}, errors.New("EMAIL_PAYLOAD_SECRET must be at least 32 bytes")
		}
	}
	if cfg.Mode == EmailModeTerminal || cfg.Mode == EmailModeConsole {
		cfg.Mode = EmailModeTerminal
		return cfg, nil
	}
	if cfg.SMTPHost == "" || cfg.SMTPFrom == "" {
		return EmailConfig{}, errors.New("SMTP_HOST and SMTP_FROM are required when EMAIL_MODE=smtp")
	}
	if (cfg.SMTPUsername == "") != (cfg.SMTPPassword == "") {
		return EmailConfig{}, errors.New("SMTP_USERNAME and SMTP_PASSWORD must be configured together")
	}
	return cfg, nil
}
