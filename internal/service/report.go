// internal/service/report.go
package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cyberoptic/openvas-tracker/internal/database/queries"
	"github.com/cyberoptic/openvas-tracker/internal/report"
)

type ReportService struct {
	q     *queries.Queries
	vulns *VulnerabilityService
}

func NewReportService(db *sql.DB, vulns *VulnerabilityService) *ReportService {
	return &ReportService{q: queries.New(db), vulns: vulns}
}

func (s *ReportService) Create(ctx context.Context, params queries.CreateReportParams) (queries.Report, error) {
	if params.ID == "" {
		params.ID = uuid.New().String()
	}
	return s.q.CreateReport(ctx, params)
}

func (s *ReportService) Get(ctx context.Context, id string) (queries.Report, error) {
	return s.q.GetReport(ctx, id)
}

func (s *ReportService) List(ctx context.Context, userID string, limit, offset int32) ([]queries.Report, error) {
	return s.q.ListReports(ctx, queries.ListReportsParams{UserID: userID, Limit: limit, Offset: offset})
}

func (s *ReportService) UpdateStatus(ctx context.Context, id string, status string, data []byte) error {
	return s.q.UpdateReportStatus(ctx, queries.UpdateReportStatusParams{
		ID:       id,
		Status:   queries.ReportStatus(status),
		FileData: data,
	})
}

func (s *ReportService) Generate(ctx context.Context, reportID string, scanIDs []string, format string, userID string) ([]byte, error) {
	// Gather vulnerabilities from all scans
	var allVulns []queries.Vulnerability
	for _, sid := range scanIDs {
		vulns, err := s.vulns.ListByScan(ctx, sid)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch vulns for scan %s: %w", sid, err)
		}
		allVulns = append(allVulns, vulns...)
	}

	// Build report data
	data := report.ReportData{
		Title:           "Vulnerability Report",
		GeneratedAt:     time.Now().Format(time.RFC3339),
		TotalVulns:      len(allVulns),
		Vulnerabilities: allVulns,
	}
	for _, v := range allVulns {
		switch v.Severity {
		case "critical":
			data.CriticalCount++
		case "high":
			data.HighCount++
		case "medium":
			data.MediumCount++
		case "low":
			data.LowCount++
		case "info":
			data.InfoCount++
		}
	}

	// Generate in requested format
	switch format {
	case "html":
		return report.GenerateHTML(data)
	case "pdf":
		return report.GeneratePDF(data)
	case "excel":
		return report.GenerateExcel(data)
	case "markdown":
		return report.GenerateMarkdown(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}
