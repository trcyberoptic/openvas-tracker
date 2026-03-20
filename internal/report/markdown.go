// internal/report/markdown.go
package report

import (
	"bytes"
	"fmt"
)

func GenerateMarkdown(data ReportData) ([]byte, error) {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "# %s\n\n", data.Title)
	fmt.Fprintf(&buf, "**Generated:** %s | **Scan:** %s\n\n", data.GeneratedAt, data.ScanName)

	fmt.Fprintf(&buf, "## Summary\n\n")
	fmt.Fprintf(&buf, "| Severity | Count |\n|----------|-------|\n")
	fmt.Fprintf(&buf, "| Critical | %d |\n", data.CriticalCount)
	fmt.Fprintf(&buf, "| High | %d |\n", data.HighCount)
	fmt.Fprintf(&buf, "| Medium | %d |\n", data.MediumCount)
	fmt.Fprintf(&buf, "| Low | %d |\n", data.LowCount)
	fmt.Fprintf(&buf, "| Info | %d |\n\n", data.InfoCount)

	fmt.Fprintf(&buf, "## Vulnerabilities\n\n")
	fmt.Fprintf(&buf, "| Severity | Title | Host | Port | CVE | CVSS |\n")
	fmt.Fprintf(&buf, "|----------|-------|------|------|-----|------|\n")

	for _, v := range data.Vulnerabilities {
		host, port, cve, cvss := "", "", "", ""
		if v.AffectedHost != nil {
			host = *v.AffectedHost
		}
		if v.AffectedPort != nil {
			port = fmt.Sprintf("%d", *v.AffectedPort)
		}
		if v.CveID != nil {
			cve = *v.CveID
		}
		if v.CvssScore != nil {
			cvss = fmt.Sprintf("%.1f", *v.CvssScore)
		}
		fmt.Fprintf(&buf, "| %s | %s | %s | %s | %s | %s |\n",
			v.Severity, v.Title, host, port, cve, cvss)
	}

	return buf.Bytes(), nil
}
