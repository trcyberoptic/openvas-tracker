// internal/report/html.go
package report

import (
	"bytes"
	"embed"
	"html/template"

	"github.com/cyberoptic/vulntrack/internal/database/queries"
)

//go:embed templates/*.html
var templateFS embed.FS

type ReportData struct {
	Title           string
	GeneratedAt     string
	ScanName        string
	TotalVulns      int
	CriticalCount   int
	HighCount       int
	MediumCount     int
	LowCount        int
	InfoCount       int
	Vulnerabilities []queries.Vulnerability
}

func GenerateHTML(data ReportData) ([]byte, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/report.html")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
