package scanner

import "io"

// Finding is a scanner-agnostic vulnerability finding.
type Finding struct {
	Host        string
	Hostname    string
	Port        string
	Protocol    string
	URL         string // Full URL path (empty for network scans)
	Parameter   string // Affected parameter (empty for network scans)
	Title       string
	Description string
	Severity    string // critical/high/medium/low/info
	CVSSScore   float64
	CVEID       string
	CWEID       string
	OID         string
	Solution    string
	Evidence    string // Proof snippet from response
	Confidence  string // confirmed/high/medium/low
	ScanType    string // "openvas" or "zap"
}

// Fingerprint returns the canonical dedup key for this finding.
// Network findings: CVE or "title:" + title
// Web findings: "cwe:" + CWE + ":url:" + URL + ":param:" + param, or title-based fallback
func (f Finding) Fingerprint() string {
	// CVE always takes priority regardless of scan type
	if f.CVEID != "" && f.CVEID != "NOCVE" {
		return f.CVEID
	}
	// Web findings with CWE + URL
	if f.URL != "" && f.CWEID != "" {
		return "cwe:" + f.CWEID + ":url:" + f.URL + ":param:" + f.Parameter
	}
	// Web findings without CWE but with URL
	if f.URL != "" {
		return "title:" + f.Title + ":url:" + f.URL
	}
	// Network findings without CVE
	return "title:" + f.Title
}

// Parser parses scanner-specific report formats into generic Findings.
type Parser interface {
	Parse(r io.Reader) ([]Finding, error)
	ScanType() string
}
