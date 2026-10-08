package service

import (
	"strings"
	"testing"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
)

func TestMailSettingsValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      MailSettings
		wantErr bool
	}{
		{"empty = off", MailSettings{}, false},
		{"host without from", MailSettings{SMTPHost: "smtp.example.com"}, true},
		{"valid", MailSettings{SMTPHost: "smtp.example.com", SMTPFrom: "Tracker <noreply@example.com>", NotifyUnassignedTo: "sec@example.com", BaseURL: "https://tracker.example.com/"}, false},
		{"bad from", MailSettings{SMTPHost: "smtp.example.com", SMTPFrom: "noreply"}, true},
		{"bad recipient", MailSettings{NotifyUnassignedTo: "sec at example"}, true},
		{"bad port", MailSettings{SMTPPort: 70000}, true},
		{"base url without scheme", MailSettings{BaseURL: "tracker.example.com"}, true},
	}
	for _, tt := range tests {
		err := tt.in.Validate()
		if (err != nil) != tt.wantErr {
			t.Errorf("%s: Validate() err = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}

	m := MailSettings{SMTPHost: " smtp.example.com ", SMTPFrom: "noreply@example.com", NotifyUnassignedTo: "Security <sec@example.com>", BaseURL: "https://t.example.com/"}
	if err := m.Validate(); err != nil {
		t.Fatal(err)
	}
	if m.SMTPPort != 587 || m.SMTPHost != "smtp.example.com" || m.NotifyUnassignedTo != "sec@example.com" || m.BaseURL != "https://t.example.com" {
		t.Errorf("not normalised: %+v", m)
	}
}

func TestImportDigest(t *testing.T) {
	low, high := 3.1, 9.8
	created := []queries.Ticket{
		{ID: "t1", Title: "[LOW] Weak thing", CvssScore: &low},
		{ID: "t2", Title: "[CRITICAL] Bad thing", CvssScore: &high},
	}
	reopened := []queries.Ticket{{ID: "t3", Title: "[MEDIUM] Back again"}}

	subject, body := importDigest("openvas", created, reopened, "https://t.example.com")
	if subject != "[OpenVAS-Tracker] Unzugewiesene Tickets: 2 neu, 1 wieder geöffnet" {
		t.Errorf("subject = %q", subject)
	}
	if strings.Index(body, "Bad thing") > strings.Index(body, "Weak thing") {
		t.Error("tickets not sorted by CVSS descending")
	}
	for _, want := range []string{"OpenVAS-Import", "Neu (2):", "Wieder geöffnet (1):", "(CVSS 9.8)", "https://t.example.com/tickets/t3"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q:\n%s", want, body)
		}
	}

	subject, body = importDigest("zap", nil, reopened, "")
	if subject != "[OpenVAS-Tracker] Unzugewiesene Tickets: 1 wieder geöffnet" || strings.Contains(body, "Neu (") || strings.Contains(body, "/tickets/") {
		t.Errorf("reopen-only digest wrong: %q\n%s", subject, body)
	}
}

func TestAssignedMail(t *testing.T) {
	one := []queries.Ticket{{ID: "t1", Title: "[HIGH] Thing"}}
	if s, b := assignedMail("alice", one, ""); s != "[OpenVAS-Tracker] Ticket zugewiesen: [HIGH] Thing" || !strings.Contains(b, "Zugewiesen von: alice") {
		t.Errorf("single: %q\n%s", s, b)
	}
	two := append(one, queries.Ticket{ID: "t2", Title: "[LOW] Other"})
	if s, _ := assignedMail("alice", two, ""); s != "[OpenVAS-Tracker] 2 Tickets zugewiesen" {
		t.Errorf("bulk subject = %q", s)
	}
}

func TestPasswordRequiredOnHostChange(t *testing.T) {
	stored := MailSettings{SMTPHost: "smtp.example.com", SMTPPassword: "secret"}
	if err := passwordRequiredOnHostChange(stored, MailSettings{SMTPHost: "relay.attacker.tld"}); err == nil {
		t.Error("host change without password accepted — the stored password would be sent to the new host")
	}
	for name, in := range map[string]MailSettings{
		"same host":           {SMTPHost: "smtp.example.com"},
		"same host, new case": {SMTPHost: "SMTP.example.com"},
		"new host + password": {SMTPHost: "relay.other.tld", SMTPPassword: "new"},
	} {
		if err := passwordRequiredOnHostChange(stored, in); err != nil {
			t.Errorf("%s: unexpected error %v", name, err)
		}
	}
	if err := passwordRequiredOnHostChange(MailSettings{}, MailSettings{SMTPHost: "smtp.example.com"}); err != nil {
		t.Errorf("no stored password: unexpected error %v", err)
	}
}
