package notification

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// TerminalSender implements Sender by printing transactional messages directly
// to stdout or a designated io.Writer. This enables convenient local development
// and verification without requiring a live SMTP server.
type TerminalSender struct {
	out io.Writer
}

func NewTerminalSender(out io.Writer) *TerminalSender {
	if out == nil {
		out = os.Stdout
	}
	return &TerminalSender{out: out}
}

func (s *TerminalSender) Send(_ context.Context, msg Message) error {
	bar := strings.Repeat("=", 64)
	fmt.Fprintf(s.out, "\n%s\n[TRANSACTIONAL EMAIL - TERMINAL MODE]\nTo: %s\nSubject: %s\n\n%s\n%s\n\n",
		bar, msg.To, msg.Subject, msg.Text, bar)
	return nil
}
