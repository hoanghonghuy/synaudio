package notification_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/notification"
)

func TestTerminalSenderOutputsMessageToWriter(t *testing.T) {
	var buf bytes.Buffer
	sender := notification.NewTerminalSender(&buf)

	msg := notification.Message{
		ID:      "msg-123",
		To:      "user@example.com",
		Subject: "Mã xác minh tài khoản",
		Text:    "Mã xác minh của bạn là: 888999\nHoặc bấm vào liên kết: https://app.example.com/verify?token=xyz",
	}

	err := sender.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected sender error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "user@example.com") {
		t.Fatalf("expected recipient in output, got: %s", output)
	}
	if !strings.Contains(output, "Mã xác minh tài khoản") {
		t.Fatalf("expected subject in output, got: %s", output)
	}
	if !strings.Contains(output, "888999") {
		t.Fatalf("expected text in output, got: %s", output)
	}
}
