package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	netmail "net/mail"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/mail"
)

// MailSettings live in app_settings and are read on every send, so edits in
// the web UI apply immediately. SMTPHost empty = mail notifications off.
type MailSettings struct {
	SMTPHost           string `json:"smtp_host"`
	SMTPPort           int    `json:"smtp_port"`
	SMTPUser           string `json:"smtp_user"`
	SMTPPassword       string `json:"smtp_password,omitempty"`
	SMTPFrom           string `json:"smtp_from"`
	NotifyUnassignedTo string `json:"notify_unassigned_to"`
	BaseURL            string `json:"base_url"`
}

const defaultSMTPPort = 587

// Validate checks and normalises the settings in place.
func (m *MailSettings) Validate() error {
	m.SMTPHost = strings.TrimSpace(m.SMTPHost)
	m.SMTPUser = strings.TrimSpace(m.SMTPUser)
	m.SMTPFrom = strings.TrimSpace(m.SMTPFrom)
	m.NotifyUnassignedTo = strings.TrimSpace(m.NotifyUnassignedTo)
	m.BaseURL = strings.TrimRight(strings.TrimSpace(m.BaseURL), "/")
	if m.SMTPPort == 0 {
		m.SMTPPort = defaultSMTPPort
	}
	if m.SMTPPort < 1 || m.SMTPPort > 65535 {
		return errors.New("smtp_port must be between 1 and 65535")
	}
	if m.SMTPHost != "" && m.SMTPFrom == "" {
		return errors.New("smtp_from is required when smtp_host is set")
	}
	if m.SMTPFrom != "" {
		if _, err := netmail.ParseAddress(m.SMTPFrom); err != nil {
			return fmt.Errorf("smtp_from: %v", err)
		}
	}
	if m.NotifyUnassignedTo != "" {
		a, err := netmail.ParseAddress(m.NotifyUnassignedTo)
		if err != nil {
			return fmt.Errorf("notify_unassigned_to: %v", err)
		}
		m.NotifyUnassignedTo = a.Address
	}
	if m.BaseURL != "" {
		u, err := url.Parse(m.BaseURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return errors.New("base_url must be an http(s) URL, e.g. https://openvas-tracker.example.com")
		}
	}
	return nil
}

func (m MailSettings) mailConfig() mail.Config {
	return mail.Config{Host: m.SMTPHost, Port: m.SMTPPort, User: m.SMTPUser, Password: m.SMTPPassword, From: m.SMTPFrom}
}

type MailNotifier struct {
	q *queries.Queries
}

func NewMailNotifier(db *sql.DB) *MailNotifier {
	return &MailNotifier{q: queries.New(db)}
}

func (n *MailNotifier) Settings(ctx context.Context) (MailSettings, error) {
	kv, err := n.q.GetAppSettings(ctx)
	if err != nil {
		return MailSettings{}, err
	}
	port, _ := strconv.Atoi(kv["smtp_port"])
	if port == 0 {
		port = defaultSMTPPort
	}
	return MailSettings{
		SMTPHost: kv["smtp_host"], SMTPPort: port, SMTPUser: kv["smtp_user"], SMTPPassword: kv["smtp_password"],
		SMTPFrom: kv["smtp_from"], NotifyUnassignedTo: kv["notify_unassigned_to"], BaseURL: kv["base_url"],
	}, nil
}

// SaveSettings stores validated settings. An empty password keeps the stored one,
// so the UI never has to hold the secret.
func (n *MailNotifier) SaveSettings(ctx context.Context, m MailSettings) error {
	if err := m.Validate(); err != nil {
		return err
	}
	vals := map[string]string{
		"smtp_host": m.SMTPHost, "smtp_port": strconv.Itoa(m.SMTPPort), "smtp_user": m.SMTPUser,
		"smtp_from": m.SMTPFrom, "notify_unassigned_to": m.NotifyUnassignedTo, "base_url": m.BaseURL,
	}
	if m.SMTPPassword != "" {
		vals["smtp_password"] = m.SMTPPassword
	}
	for k, v := range vals {
		if err := n.q.SetAppSetting(ctx, k, v); err != nil {
			return err
		}
	}
	return nil
}

// SendTest sends a test mail synchronously and returns the relay's error verbatim.
func (n *MailNotifier) SendTest(ctx context.Context, to string) error {
	m, err := n.Settings(ctx)
	if err != nil {
		return err
	}
	if m.SMTPHost == "" {
		return errors.New("SMTP is not configured")
	}
	a, err := netmail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("recipient: %v", err)
	}
	return mail.Send(m.mailConfig(), []string{a.Address}, "[OpenVAS-Tracker] Testmail",
		"Diese Testmail bestätigt, dass der OpenVAS-Tracker über "+m.SMTPHost+" Mails verschicken kann.\n")
}

// NotifyImport mails the unassigned tickets an import created or reopened to
// notify_unassigned_to. Meant to run in its own goroutine after the commit.
func (n *MailNotifier) NotifyImport(scanType string, createdIDs, reopenedIDs []string) {
	if len(createdIDs) == 0 && len(reopenedIDs) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	m, err := n.Settings(ctx)
	if err != nil {
		log.Printf("mail: load settings: %v", err)
		return
	}
	if m.SMTPHost == "" || m.NotifyUnassignedTo == "" {
		return
	}
	subject, body := importDigest(scanType, n.tickets(ctx, createdIDs), n.tickets(ctx, reopenedIDs), m.BaseURL)
	if err := mail.Send(m.mailConfig(), []string{m.NotifyUnassignedTo}, subject, body); err != nil {
		log.Printf("mail: import digest to %s failed: %v", m.NotifyUnassignedTo, err)
	}
}

// NotifyAssigned mails the assignee about tickets assigned in one action.
// Self-assignment, unassignment and users who opted out get no mail.
// Meant to run in its own goroutine.
func (n *MailNotifier) NotifyAssigned(actorID, assigneeID string, ticketIDs []string) {
	if assigneeID == "" || assigneeID == actorID || len(ticketIDs) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	m, err := n.Settings(ctx)
	if err != nil {
		log.Printf("mail: load settings: %v", err)
		return
	}
	if m.SMTPHost == "" {
		return
	}
	assignee, err := n.q.GetUserByID(ctx, assigneeID)
	if err != nil || assignee.Email == "" || !assignee.IsActive {
		return
	}
	if on, err := n.q.GetUserEmailNotifications(ctx, assigneeID); err != nil || !on {
		return
	}
	actor := "Unbekannt"
	if u, err := n.q.GetUserByID(ctx, actorID); err == nil {
		actor = u.Username
	}
	tickets := n.tickets(ctx, ticketIDs)
	if len(tickets) == 0 {
		return
	}
	subject, body := assignedMail(actor, tickets, m.BaseURL)
	if err := mail.Send(m.mailConfig(), []string{assignee.Email}, subject, body); err != nil {
		log.Printf("mail: assignment mail to %s failed: %v", assignee.Email, err)
	}
}

func (n *MailNotifier) tickets(ctx context.Context, ids []string) []queries.Ticket {
	out := make([]queries.Ticket, 0, len(ids))
	for _, id := range ids {
		if t, err := n.q.GetTicket(ctx, id); err == nil {
			out = append(out, t)
		}
	}
	return out
}

const mailFooter = "\n--\nDiese Mail wurde automatisch vom OpenVAS-Tracker verschickt.\n"

func importDigest(scanType string, created, reopened []queries.Ticket, baseURL string) (subject, body string) {
	var parts []string
	if len(created) > 0 {
		parts = append(parts, fmt.Sprintf("%d neu", len(created)))
	}
	if len(reopened) > 0 {
		parts = append(parts, fmt.Sprintf("%d wieder geöffnet", len(reopened)))
	}
	subject = "[OpenVAS-Tracker] Unzugewiesene Tickets: " + strings.Join(parts, ", ")

	var b strings.Builder
	fmt.Fprintf(&b, "Der %s-Import hat folgende Tickets ohne Zuweisung hinterlassen.\n", scanTypeLabel(scanType))
	writeTicketSection(&b, "Neu", created, baseURL)
	writeTicketSection(&b, "Wieder geöffnet", reopened, baseURL)
	b.WriteString(mailFooter)
	return subject, b.String()
}

func assignedMail(actor string, tickets []queries.Ticket, baseURL string) (subject, body string) {
	if len(tickets) == 1 {
		subject = "[OpenVAS-Tracker] Ticket zugewiesen: " + tickets[0].Title
	} else {
		subject = fmt.Sprintf("[OpenVAS-Tracker] %d Tickets zugewiesen", len(tickets))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Zugewiesen von: %s\n", actor)
	writeTicketSection(&b, "Tickets", tickets, baseURL)
	b.WriteString(mailFooter)
	b.WriteString("Zuweisungs-Mails lassen sich unter Settings > Profile abschalten.\n")
	return subject, b.String()
}

// writeTicketSection lists tickets highest CVSS first. The ticket title already
// carries severity and host.
func writeTicketSection(b *strings.Builder, heading string, tickets []queries.Ticket, baseURL string) {
	if len(tickets) == 0 {
		return
	}
	sorted := append([]queries.Ticket(nil), tickets...)
	sort.SliceStable(sorted, func(i, j int) bool { return cvss(sorted[i]) > cvss(sorted[j]) })
	fmt.Fprintf(b, "\n%s (%d):\n", heading, len(sorted))
	for _, t := range sorted {
		fmt.Fprintf(b, "- %s", t.Title)
		if t.CvssScore != nil {
			fmt.Fprintf(b, " (CVSS %.1f)", *t.CvssScore)
		}
		b.WriteString("\n")
		if baseURL != "" {
			fmt.Fprintf(b, "  %s/tickets/%s\n", baseURL, t.ID)
		}
	}
}

func cvss(t queries.Ticket) float64 {
	if t.CvssScore == nil {
		return 0
	}
	return *t.CvssScore
}

func scanTypeLabel(scanType string) string {
	switch scanType {
	case "openvas":
		return "OpenVAS"
	case "zap":
		return "ZAP"
	}
	return scanType
}
