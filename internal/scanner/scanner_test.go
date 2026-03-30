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
			name:     "web finding with CWE and URL",
			finding:  Finding{CWEID: "79", URL: "/app/search", Parameter: "q", ScanType: "zap"},
			expected: "cwe:79:url:/app/search:param:q",
		},
		{
			name:     "web finding with CWE, URL, no param",
			finding:  Finding{CWEID: "352", URL: "/app/transfer", ScanType: "zap"},
			expected: "cwe:352:url:/app/transfer:param:",
		},
		{
			name:     "web finding without CWE",
			finding:  Finding{Title: "Information Disclosure", URL: "/api/debug", ScanType: "zap"},
			expected: "title:Information Disclosure:url:/api/debug",
		},
		{
			name:     "web finding with CVE takes CVE path",
			finding:  Finding{CVEID: "CVE-2024-9999", CWEID: "79", URL: "/app", ScanType: "zap"},
			expected: "CVE-2024-9999",
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
