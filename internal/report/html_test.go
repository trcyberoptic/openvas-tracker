// internal/report/html_test.go
package report

import (
	"strings"
	"testing"
)

func TestGenerateHTML(t *testing.T) {
	data := ReportData{
		Title:         "Test Report",
		GeneratedAt:   "2026-03-20",
		TotalVulns:    2,
		CriticalCount: 1,
		HighCount:     1,
	}

	output, err := GenerateHTML(data)
	if err != nil {
		t.Fatalf("GenerateHTML error: %v", err)
	}
	html := string(output)
	if !strings.Contains(html, "Test Report") {
		t.Error("expected report title in output")
	}
	if !strings.Contains(html, "2026-03-20") {
		t.Error("expected generated date in output")
	}
	if !strings.Contains(html, "<table>") {
		t.Error("expected table element in output")
	}
}
