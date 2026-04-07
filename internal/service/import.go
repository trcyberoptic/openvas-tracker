package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"net"
	"encoding/hex"
	"os"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cyberoptic/openvas-tracker/internal/auth"
	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/scanner"
)

// ImportResult contains the statistics from a single import operation.
type ImportResult struct {
	ScanID              string `json:"scan_id"`
	VulnsImported       int    `json:"vulnerabilities_imported"`
	VulnsSkipped        int    `json:"vulnerabilities_skipped"`
	TicketsCreated      int    `json:"tickets_created"`
	TicketsReopened     int    `json:"tickets_reopened"`
	TicketsAutoResolved int    `json:"tickets_auto_resolved"`
}

type ImportService struct {
	db           *sql.DB
	q            *queries.Queries
	systemUserID string
	mu           sync.Mutex
	initDone     bool
}

func NewImportService(db *sql.DB) *ImportService {
	return &ImportService{db: db, q: queries.New(db)}
}

// autoResolveThreshold returns the configured number of consecutive scan misses
// required before a ticket is auto-resolved. Re-reads .env on each call so
// Settings page changes take effect without restart.
func autoResolveThreshold() int {
	if v := os.Getenv("OT_AUTORESOLVE_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			return n
		}
	}
	return 3
}

// BackfillHostnames resolves PTR records for all vulnerabilities missing a hostname.
func (s *ImportService) BackfillHostnames(ctx context.Context) (int, error) {
	hosts, err := s.q.DistinctHostsWithoutHostname(ctx)
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, ip := range hosts {
		hostname := resolveHostname(ip, "")
		if hostname == "" {
			continue
		}
		if err := s.q.SetHostnameByIP(ctx, ip, hostname); err != nil {
			log.Printf("backfill: failed to set hostname for %s: %v", ip, err)
			continue
		}
		log.Printf("backfill: %s → %s", ip, hostname)
		updated++
	}
	return updated, nil
}

// Import processes parsed scanner findings: creates scan, vulns, tickets in a single transaction.
// If meta is non-nil, the scan timestamps are taken from the report instead of using time.Now().
func (s *ImportService) Import(ctx context.Context, results []scanner.Finding, scanType string, meta *scanner.ScanMeta) (*ImportResult, error) {
	if err := s.resolveSystemUser(ctx); err != nil {
		return nil, fmt.Errorf("resolve system user: %w", err)
	}

	now := time.Now()
	scanID := uuid.New().String()

	// Use report timestamps when available, fall back to current time
	startedAt := now
	completedAt := now
	if meta != nil && meta.StartedAt != nil {
		startedAt = *meta.StartedAt
	}
	if meta != nil && meta.CompletedAt != nil {
		completedAt = *meta.CompletedAt
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	tq := queries.New(tx)

	scan, err := tq.CreateScan(ctx, queries.CreateScanParams{
		ID:       scanID,
		Name:     fmt.Sprintf("%s Import %s", strings.ToUpper(scanType), startedAt.Format("2006-01-02 15:04:05")),
		ScanType: queries.ScanType(scanType),
		Status:   queries.ScanStatusCompleted,
		UserID:   s.systemUserID,
	})
	if err != nil {
		return nil, fmt.Errorf("create scan: %w", err)
	}

	if _, err := tq.UpdateScanStatus(ctx, queries.UpdateScanStatusParams{
		ID: scan.ID, Status: queries.ScanStatusCompleted, StartedAt: &startedAt, CompletedAt: &completedAt,
	}); err != nil {
		log.Printf("import: failed to update scan status: %v", err)
	}

	res := &ImportResult{ScanID: scan.ID}

	// Collect all distinct hosts from the raw results (before info-filter)
	// so we know which hosts were in scope for this scan.
	scannedHosts := make(map[string]bool)
	for _, r := range results {
		if r.Host != "" {
			scannedHosts[r.Host] = true
		}
	}
	for host := range scannedHosts {
		if err := tq.InsertScanHost(ctx, scan.ID, host); err != nil {
			log.Printf("import: failed to record scan host %s: %v", host, err)
		}
	}

	for _, r := range results {
		port, proto := parsePort(r.Port)
		severity := mapSeverity(r.Severity, r.CVSSScore)

		if severity == "info" && r.CVSSScore == 0 {
			res.VulnsSkipped++
			continue
		}

		vulnID := uuid.New().String()
		_, err := tq.CreateVulnerability(ctx, queries.CreateVulnerabilityParams{
			ID:             vulnID,
			ScanID:         scan.ID,
			UserID:         s.systemUserID,
			Title:          r.Title,
			Description:    strPtr(r.Description),
			Severity:       queries.SeverityLevel(severity),
			CvssScore:      f64Ptr(r.CVSSScore),
			CveID:          cvePtr(r.CVEID),
			CweID:          strPtr(r.CWEID),
			AffectedHost:   strPtr(r.Host),
			Hostname:       strPtr(resolveHostname(r.Host, r.Hostname)),
			URL:            strPtr(r.URL),
			Parameter:      strPtr(r.Parameter),
			AffectedPort:   port,
			Protocol:       proto,
			Solution:       strPtr(r.Solution),
			Evidence:       strPtr(r.Evidence),
			Confidence:     strPtr(r.Confidence),
			VulnReferences: []byte("[]"),
		})
		if err != nil {
			log.Printf("import: failed to create vuln %q for host %s: %v", r.Title, r.Host, err)
			res.VulnsSkipped++
			continue
		}
		res.VulnsImported++

		created, reopened := s.processTicket(ctx, tq, r, vulnID, severity, now)
		if created {
			res.TicketsCreated++
		}
		if reopened {
			res.TicketsReopened++
		}
	}

	s.reopenExpiredRiskAccepted(ctx, tq)
	res.TicketsAutoResolved = s.autoResolveStale(ctx, tq, scan.ID, scanType)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return res, nil
}

func (s *ImportService) processTicket(ctx context.Context, q *queries.Queries, r scanner.Finding, vulnID, severity string, now time.Time) (created, reopened bool) {
	if r.Host == "" {
		return false, false
	}

	existing, err := q.FindTicketByFingerprint(ctx, r.Host, r.CVEID, r.Title)
	if err != nil {
		return s.createTicket(ctx, q, r, vulnID, severity), false
	}

	oldStatus := string(existing.Status)

	switch existing.Status {
	case queries.TicketStatusFalsePositive:
		return false, false

	case queries.TicketStatusRiskAccepted:
		// Check if an active auto-accept rule still matches — if so, keep accepted
		fp := r.Fingerprint()
		if _, err := q.MatchRiskAcceptRule(ctx, fp, r.Host); err == nil {
			if err := q.TouchTicket(ctx, queries.TouchTicketParams{
				ID: existing.ID, VulnerabilityID: vulnID,
			}); err != nil {
				log.Printf("import: failed to touch ticket %s: %v", existing.ID, err)
			}
			note := fmt.Sprintf("Finding still present in scan, kept risk-accepted by rule. CVE: %s, Host: %s", r.CVEID, r.Host)
			logActivity(ctx, q, existing.ID, "still_present", nil, nil, "Automatic", &note)
			return false, false
		}
		// No active rule — reopen
		if err := q.ReopenTicket(ctx, queries.ReopenTicketParams{
			ID: existing.ID, VulnerabilityID: vulnID,
		}); err != nil {
			return false, false
		}
		newStatus := "open"
		note := fmt.Sprintf("Finding reappeared in scan — reopened (no active rule). CVE: %s, Host: %s", r.CVEID, r.Host)
		logActivity(ctx, q, existing.ID, "status_changed", &oldStatus, &newStatus, "Automatic", &note)
		return false, true

	case queries.TicketStatusFixed:
		if err := q.ReopenTicket(ctx, queries.ReopenTicketParams{
			ID: existing.ID, VulnerabilityID: vulnID,
		}); err != nil {
			return false, false
		}
		newStatus := "open"
		note := fmt.Sprintf("Finding reappeared in scan — reopened. CVE: %s, Host: %s", r.CVEID, r.Host)
		logActivity(ctx, q, existing.ID, "status_changed", &oldStatus, &newStatus, "Automatic", &note)
		return false, true

	case queries.TicketStatusPendingResolution:
		// Finding reappeared while in pending_resolution — reset counter, go back to open
		if err := q.ReopenTicket(ctx, queries.ReopenTicketParams{
			ID: existing.ID, VulnerabilityID: vulnID,
		}); err != nil {
			log.Printf("import: failed to reopen pending_resolution ticket %s: %v", existing.ID, err)
			return false, false
		}
		newStatus := "open"
		note := fmt.Sprintf("Finding reappeared after %d/%d misses — counter reset. CVE: %s, Host: %s",
			existing.ConsecutiveMisses, autoResolveThreshold(), r.CVEID, r.Host)
		logActivity(ctx, q, existing.ID, "status_changed", &oldStatus, &newStatus, "Automatic", &note)
		return false, true

	default: // open
		if err := q.TouchTicket(ctx, queries.TouchTicketParams{
			ID: existing.ID, VulnerabilityID: vulnID,
		}); err != nil {
			log.Printf("import: failed to touch ticket %s: %v", existing.ID, err)
		}
		// Reset any accumulated misses (e.g. from a previous flap cycle that was manually reopened)
		if existing.ConsecutiveMisses > 0 {
			if err := q.ResetConsecutiveMisses(ctx, existing.ID); err != nil {
				log.Printf("import: failed to reset misses for ticket %s: %v", existing.ID, err)
			}
		}
		note := fmt.Sprintf("Finding still present in scan. CVE: %s, Host: %s", r.CVEID, r.Host)
		logActivity(ctx, q, existing.ID, "still_present", nil, nil, "Automatic", &note)
		return false, false
	}
}

func (s *ImportService) createTicket(ctx context.Context, q *queries.Queries, r scanner.Finding, vulnID, severity string) bool {
	priority := mapSeverityToPriority(severity)
	title := fmt.Sprintf("[%s] %s — %s", strings.ToUpper(severity), r.Title, r.Host)
	var desc *string
	if r.Description != "" {
		d := fmt.Sprintf("%s\n\nSolution: %s", r.Description, r.Solution)
		desc = &d
	}

	ticketID := uuid.New().String()
	_, err := q.CreateTicket(ctx, queries.CreateTicketParams{
		ID: ticketID, Title: title, Description: desc,
		Priority: queries.TicketPriority(priority), VulnerabilityID: &vulnID, CreatedBy: s.systemUserID,
	})
	if err != nil {
		return false
	}

	// Check if a risk accept rule matches this finding
	fp := r.Fingerprint()
	if rule, err := q.MatchRiskAcceptRule(ctx, fp, r.Host); err == nil {
		q.UpdateTicketStatus(ctx, queries.UpdateTicketStatusParams{ID: ticketID, Status: queries.TicketStatusRiskAccepted})
		if rule.ExpiresAt != nil {
			q.SetRiskAcceptedUntil(ctx, ticketID, rule.ExpiresAt)
		}
		newStatus := "risk_accepted"
		note := fmt.Sprintf("Auto risk-accepted by rule: %s", rule.Reason)
		logActivity(ctx, q, ticketID, "status_changed", nil, &newStatus, "Automatic", &note)
	}

	if err := q.TouchTicket(ctx, queries.TouchTicketParams{ID: ticketID, VulnerabilityID: vulnID}); err != nil {
		log.Printf("import: failed to touch new ticket %s: %v", ticketID, err)
	}

	newStatus := "open"
	note := fmt.Sprintf("Ticket created from %s import. CVE: %s, Host: %s, CVSS: %.1f", r.ScanType, r.CVEID, r.Host, r.CVSSScore)
	logActivity(ctx, q, ticketID, "created", nil, &newStatus, "Automatic", &note)
	return true
}

func (s *ImportService) reopenExpiredRiskAccepted(ctx context.Context, q *queries.Queries) {
	reopened, err := q.ReopenExpiredRiskAccepted(ctx)
	if err != nil {
		log.Printf("reopen expired risk_accepted error: %v", err)
		return
	}
	for _, t := range reopened {
		oldStatus := "risk_accepted"
		newStatus := "open"
		note := "Risk acceptance expired — ticket reopened"
		logActivity(ctx, q, t.ID, "status_changed", &oldStatus, &newStatus, "Automatic", &note)
	}
}

func (s *ImportService) autoResolveStale(ctx context.Context, q *queries.Queries, scanID string, scanType string) int {
	threshold := autoResolveThreshold()

	staleTickets, err := q.IncrementMissesForStaleTickets(ctx, scanID, scanType)
	if err != nil {
		log.Printf("auto-resolve error: %v", err)
		return 0
	}

	resolved := 0
	for _, t := range staleTickets {
		oldStatus := string(t.Status)
		misses := t.ConsecutiveMisses // already incremented by the query

		if misses >= threshold {
			// Threshold reached — resolve
			if err := q.ResolveTicket(ctx, t.ID); err != nil {
				log.Printf("auto-resolve: failed to resolve ticket %s: %v", t.ID, err)
				continue
			}
			newStatus := "fixed"
			note := fmt.Sprintf("Finding not present in %d consecutive scans — auto-fixed", misses)
			logActivity(ctx, q, t.ID, "status_changed", &oldStatus, &newStatus, "Automatic", &note)
			resolved++
		} else if t.Status == queries.TicketStatusOpen {
			// First miss — transition to pending_resolution
			if err := q.SetTicketPendingResolution(ctx, t.ID); err != nil {
				log.Printf("auto-resolve: failed to set pending_resolution for ticket %s: %v", t.ID, err)
				continue
			}
			newStatus := "pending_resolution"
			note := fmt.Sprintf("Finding not present in scan — pending resolution (%d/%d consecutive misses)", misses, threshold)
			logActivity(ctx, q, t.ID, "status_changed", &oldStatus, &newStatus, "Automatic", &note)
		} else {
			// Already pending_resolution, just log the miss count
			note := fmt.Sprintf("Finding not present in scan (%d/%d consecutive misses)", misses, threshold)
			logActivity(ctx, q, t.ID, "still_present", nil, nil, "Automatic", &note)
		}
	}
	return resolved
}

func (s *ImportService) resolveSystemUser(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.initDone {
		return nil
	}

	user, err := s.q.GetUserByUsername(ctx, "openvas-import")
	if err == nil {
		s.systemUserID = user.ID
		s.initDone = true
		return nil
	}

	randBytes := make([]byte, 32)
	if _, err := rand.Read(randBytes); err != nil {
		return fmt.Errorf("generate random password: %w", err)
	}
	hash, err := auth.HashPassword(hex.EncodeToString(randBytes))
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user, err = s.q.CreateUser(ctx, queries.CreateUserParams{
		ID: uuid.New().String(), Email: "openvas-import@system.local",
		Username: "openvas-import", Password: hash, Role: queries.UserRoleViewer,
	})
	if err != nil {
		user, err = s.q.GetUserByUsername(ctx, "openvas-import")
		if err != nil {
			return fmt.Errorf("resolve system user: %w", err)
		}
	}

	s.systemUserID = user.ID
	s.initDone = true
	return nil
}

// Helpers

func logActivity(ctx context.Context, q *queries.Queries, ticketID, action string, oldVal, newVal *string, changedBy string, note *string) {
	q.LogTicketActivity(ctx, queries.LogTicketActivityParams{
		ID: uuid.New().String(), TicketID: ticketID, Action: action,
		OldValue: oldVal, NewValue: newVal, ChangedBy: changedBy, Note: note,
	})
}

// resolveHostname returns the hostname from XML, or falls back to PTR lookup.
// Normalizes: hostname part uppercase, domain part lowercase.
func resolveHostname(ip, xmlHostname string) string {
	raw := xmlHostname
	if raw == "" {
		if ip == "" {
			return ""
		}
		names, err := net.LookupAddr(ip)
		if err != nil || len(names) == 0 {
			return ""
		}
		raw = strings.TrimSuffix(names[0], ".")
	}
	return normalizeHostname(raw)
}

// normalizeHostname: hostname part UPPERCASE, domain part lowercase.
// e.g. "server01.example.com" → "SERVER01.example.com"
func normalizeHostname(h string) string {
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, ".", 2)
	result := strings.ToUpper(parts[0])
	if len(parts) == 2 {
		result += "." + strings.ToLower(parts[1])
	}
	return result
}

// VulnFingerprint returns the canonical fingerprint for a vulnerability: CVE if available, otherwise "title:" + title.
func VulnFingerprint(cve, title string) string {
	if cve != "" && cve != "NOCVE" {
		return cve
	}
	return "title:" + title
}

func vulnFingerprint(cve, title string) string {
	return VulnFingerprint(cve, title)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func f64Ptr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

func cvePtr(s string) *string {
	if s == "" || s == "NOCVE" {
		return nil
	}
	return &s
}

func mapSeverity(threat string, cvss float64) string {
	switch strings.ToLower(threat) {
	case "high":
		if cvss >= 9.0 {
			return "critical"
		}
		return "high"
	case "medium":
		return "medium"
	case "low":
		return "low"
	default:
		// Fallback: derive from CVSS when threat field is missing/unexpected
		if cvss >= 9.0 {
			return "critical"
		}
		if cvss >= 7.0 {
			return "high"
		}
		if cvss >= 4.0 {
			return "medium"
		}
		if cvss > 0 {
			return "low"
		}
		return "info"
	}
}

func mapSeverityToPriority(severity string) string {
	switch severity {
	case "critical":
		return "critical"
	case "high":
		return "high"
	case "medium":
		return "medium"
	default:
		return "low"
	}
}

func parsePort(portStr string) (*int32, *string) {
	if portStr == "" {
		return nil, nil
	}
	parts := strings.SplitN(portStr, "/", 2)
	if len(parts) != 2 {
		return nil, nil
	}
	num, err := strconv.ParseInt(parts[0], 10, 32)
	if err != nil {
		return nil, nil
	}
	port := int32(num)
	proto := parts[1]
	return &port, &proto
}
