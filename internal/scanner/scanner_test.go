package scanner

import "testing"

func TestFindingFingerprint(t *testing.T) {
	tests := []struct {
		name     string
		finding  Finding
		expected string
	}{
		{
			name:     "network finding with CVE",
			finding:  Finding{CVEID: "CVE-2024-0001", Title: "SSL Expired", ScanType: "openvas"},
			expected: "CVE-2024-0001",
		},
		{
			name:     "network finding without CVE",
			finding:  Finding{Title: "SSL Expired", ScanType: "openvas"},
			expected: "title:SSL Expired",
		},
		{
			name:     "web finding with CWE, URL, and param — URL-granular",
			finding:  Finding{CWEID: "79", URL: "/app/search", Parameter: "q", ScanType: "zap"},
			expected: "cwe:79:url:/app/search:param:q",
		},
		{
			name:     "web finding with CWE, URL, no param — server-wide, one per host",
			finding:  Finding{CWEID: "693", URL: "/sitemap.xml", ScanType: "zap"},
			expected: "cwe:693",
		},
		{
			name:     "web finding without CWE, with param",
			finding:  Finding{Title: "Information Disclosure", URL: "/api/debug", Parameter: "verbose", ScanType: "zap"},
			expected: "title:Information Disclosure:url:/api/debug:param:verbose",
		},
		{
			name:     "web finding without CWE, no param — server-wide",
			finding:  Finding{Title: "Missing Header X", URL: "/robots.txt", ScanType: "zap"},
			expected: "title:Missing Header X",
		},
		{
			name:     "web finding with CVE takes CVE path",
			finding:  Finding{CVEID: "CVE-2024-9999", CWEID: "79", URL: "/app", Parameter: "q", ScanType: "zap"},
			expected: "CVE-2024-9999",
		},
		{
			name:     "same header finding different URLs — same fingerprint",
			finding:  Finding{CWEID: "693", URL: "/", Title: "CSP Header Not Set", ScanType: "zap"},
			expected: "cwe:693",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.finding.Fingerprint()
			if got != tt.expected {
				t.Errorf("Fingerprint() = %q, want %q", got, tt.expected)
			}
		})
	}
}
