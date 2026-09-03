package providers

import (
	"fmt"

	"github.com/synaudio/synaudio/backend/internal/notification"
	"github.com/synaudio/synaudio/backend/internal/platform/config"
)

func BuildEmail(cfg config.EmailConfig, store notification.OutboxStore) (*notification.Service, error) {
	if cfg.Mode == config.EmailModeDisabled {
		return nil, nil
	}
	if cfg.Mode != config.EmailModeSMTP {
		return nil, fmt.Errorf("unsupported EMAIL_MODE %q", cfg.Mode)
	}
	sender, err := notification.NewSMTPSender(notification.SMTPConfig{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
		Timeout:  cfg.SMTPTimeout,
	})
	if err != nil {
		return nil, err
	}
	return notification.NewService(store, sender, cfg.PayloadSecret, cfg.AppPublicURL)
}
