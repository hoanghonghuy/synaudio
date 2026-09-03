package notification

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	PurposeEmailVerification = "EMAIL_VERIFICATION"
	PurposePasswordReset     = "PASSWORD_RESET"
)

var ErrNoPendingDelivery = errors.New("no pending email delivery")

type OutboxItem struct {
	ID               string
	Purpose          string
	RecipientEmail   string
	EncryptedPayload []byte
	AttemptCount     int
	MaxAttempts      int
}

type OutboxStore interface {
	Enqueue(ctx context.Context, item OutboxItem) error
	Claim(ctx context.Context, now, staleBefore time.Time) (OutboxItem, error)
	MarkDelivered(ctx context.Context, id string, now time.Time) error
	MarkFailed(ctx context.Context, id string, now, retryAt time.Time, failure string) error
}

type Message struct {
	To      string
	Subject string
	Text    string
}

type Sender interface {
	Send(ctx context.Context, message Message) error
}

type encryptedPayload struct {
	Link string `json:"link"`
}

type Service struct {
	store        OutboxStore
	sender       Sender
	sealer       cipher.AEAD
	appPublicURL string
	now          func() time.Time
}

func NewService(store OutboxStore, sender Sender, payloadSecret, appPublicURL string) (*Service, error) {
	payloadSecret = strings.TrimSpace(payloadSecret)
	if len(payloadSecret) < 32 {
		return nil, errors.New("EMAIL_PAYLOAD_SECRET must be at least 32 bytes")
	}
	appPublicURL = strings.TrimRight(strings.TrimSpace(appPublicURL), "/")
	if appPublicURL == "" {
		return nil, errors.New("APP_PUBLIC_URL is required for transactional email")
	}
	key := sha256.Sum256([]byte(payloadSecret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	sealer, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Service{store: store, sender: sender, sealer: sealer, appPublicURL: appPublicURL, now: time.Now}, nil
}

func (s *Service) QueueVerification(ctx context.Context, email, token string) error {
	link := s.appPublicURL + "/verify-email?email=" + url.QueryEscape(email) + "&token=" + url.QueryEscape(token)
	return s.enqueue(ctx, PurposeEmailVerification, email, link)
}

func (s *Service) QueuePasswordReset(ctx context.Context, email, token string) error {
	link := s.appPublicURL + "/reset-password?email=" + url.QueryEscape(email) + "&token=" + url.QueryEscape(token)
	return s.enqueue(ctx, PurposePasswordReset, email, link)
}

func (s *Service) enqueue(ctx context.Context, purpose, recipient, link string) error {
	payload, err := json.Marshal(encryptedPayload{Link: link})
	if err != nil {
		return err
	}
	nonce := make([]byte, s.sealer.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	sealed := s.sealer.Seal(nil, nonce, payload, []byte(purpose))
	ciphertext := append(nonce, sealed...)
	return s.store.Enqueue(ctx, OutboxItem{
		ID:               uuid.NewString(),
		Purpose:          purpose,
		RecipientEmail:   strings.TrimSpace(recipient),
		EncryptedPayload: ciphertext,
		MaxAttempts:      8,
	})
}

func (s *Service) DeliverNext(ctx context.Context) (bool, error) {
	now := s.now().UTC()
	item, err := s.store.Claim(ctx, now, now.Add(-5*time.Minute))
	if err != nil {
		if errors.Is(err, ErrNoPendingDelivery) {
			return false, nil
		}
		return false, err
	}
	payload, err := s.open(item)
	if err != nil {
		_ = s.store.MarkFailed(ctx, item.ID, now, now.Add(backoff(item.AttemptCount)), "encrypted payload could not be opened")
		return true, err
	}
	message, err := render(item.Purpose, item.RecipientEmail, payload.Link)
	if err != nil {
		_ = s.store.MarkFailed(ctx, item.ID, now, now.Add(backoff(item.AttemptCount)), "unsupported email purpose")
		return true, err
	}
	if err := s.sender.Send(ctx, message); err != nil {
		failure := sanitizeFailure(err)
		_ = s.store.MarkFailed(ctx, item.ID, now, now.Add(backoff(item.AttemptCount)), failure)
		return true, err
	}
	if err := s.store.MarkDelivered(ctx, item.ID, now); err != nil {
		return true, err
	}
	return true, nil
}

func (s *Service) open(item OutboxItem) (encryptedPayload, error) {
	nonceSize := s.sealer.NonceSize()
	if len(item.EncryptedPayload) <= nonceSize {
		return encryptedPayload{}, errors.New("invalid encrypted payload")
	}
	plain, err := s.sealer.Open(nil, item.EncryptedPayload[:nonceSize], item.EncryptedPayload[nonceSize:], []byte(item.Purpose))
	if err != nil {
		return encryptedPayload{}, err
	}
	var payload encryptedPayload
	if err := json.Unmarshal(plain, &payload); err != nil {
		return encryptedPayload{}, err
	}
	if strings.TrimSpace(payload.Link) == "" {
		return encryptedPayload{}, errors.New("empty email link")
	}
	return payload, nil
}

func render(purpose, recipient, link string) (Message, error) {
	switch purpose {
	case PurposeEmailVerification:
		return Message{To: recipient, Subject: "Verify your Synaudio email", Text: "Verify your email by opening this link:\n\n" + link + "\n"}, nil
	case PurposePasswordReset:
		return Message{To: recipient, Subject: "Reset your Synaudio password", Text: "Reset your password by opening this link:\n\n" + link + "\n"}, nil
	default:
		return Message{}, fmt.Errorf("unsupported email purpose %q", purpose)
	}
}

func backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 8 {
		attempt = 8
	}
	return time.Duration(1<<uint(attempt-1)) * time.Minute
}

func sanitizeFailure(err error) string {
	if err == nil {
		return "delivery failed"
	}
	value := strings.TrimSpace(err.Error())
	if len(value) > 300 {
		value = value[:300]
	}
	return value
}
