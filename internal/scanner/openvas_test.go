// internal/scanner/openvas_test.go
package scanner

import (
	"strings"
	"testing"
)

func TestParseOpenVASXML(t *testing.T) {
	xml := `<?xml version="1.0"?>
<report>
  <results>
    <result>
      <name>SSL/TLS Certificate Expired</name>
      <host>192.168.1.10</host>
      <port>443/tcp</port>
      <threat>High</threat>
      <severity>7.5</severity>
      <description>The SSL certificate has expired.</description>
      <nvt oid="1.3.6.1.4.1.25623.1.0.103955">
        <name>SSL/TLS Certificate Expired</name>
        <cvss_base>7.5</cvss_base>
        <cve>CVE-2024-0001</cve>
        <solution type="VendorFix">Renew the certificate.</solution>
      </nvt>
    </result>
  </results>
</report>`

	results, _, err := ParseOpenVASXML(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseOpenVASXML error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Title != "SSL/TLS Certificate Expired" {
		t.Errorf("expected title SSL/TLS Certificate Expired, got %s", r.Title)
	}
	if r.Severity != "High" {
		t.Errorf("expected severity High, got %s", r.Severity)
	}
	if r.CVSSScore != 7.5 {
		t.Errorf("expected CVSS 7.5, got %f", r.CVSSScore)
	}
	if r.Host != "192.168.1.10" {
		t.Errorf("expected host 192.168.1.10, got %s", r.Host)
	}
	if r.CVEID != "CVE-2024-0001" {
		t.Errorf("expected CVEID CVE-2024-0001, got %s", r.CVEID)
	}
	if r.ScanType != "openvas" {
		t.Errorf("expected ScanType openvas, got %s", r.ScanType)
	}
	if r.URL != "" {
		t.Errorf("expected empty URL, got %s", r.URL)
	}
}
