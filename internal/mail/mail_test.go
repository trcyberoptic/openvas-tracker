package mail

import (
	"strings"
	"testing"
	"time"
)

func TestBuildMessage(t *testing.T) {
	msg := string(buildMessage("Tracker <noreply@example.com>", []string{"a@example.com"},
		"Ticket zugewiesen: Lücke", "Zeile 1\nZeile 2\n", time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)))

	for _, want := range []string{
		"From: Tracker <noreply@example.com>\r\n",
		"To: a@example.com\r\n",
		"Subject: =?utf-8?q?Ticket_zugewiesen:_L=C3=BCcke?=\r\n",
		"Content-Type: text/plain; charset=utf-8\r\n",
		"\r\n\r\nZeile 1\r\nZeile 2\r\n",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q:\n%s", want, msg)
		}
	}
	if strings.Contains(strings.ReplaceAll(msg, "\r\n", ""), "\n") {
		t.Error("bare LF in message; SMTP needs CRLF")
	}
}
