package scanner

import (
	"strings"
	"testing"
)

// The OID is ticket identity — see queries.FindTicketByFingerprint. Both parsers
// extracted it long before anything read it, and that gap is how a feed change
// that merely dropped CVE references could split 136 tickets in production.
// If either parser stops carrying it, dedup silently falls back to the titles
// and CVEs that move around, so guard the wiring rather than the parsing.
func TestParsersCarryOID(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<report><results><result>
  <name>SSL/TLS: Renegotiation DoS Vulnerability</name>
  <host>10.10.10.6</host>
  <port>443/tcp</port>
  <threat>Medium</threat>
  <severity>5.0</severity>
  <nvt oid="1.3.6.1.4.1.25623.1.0.117761">
    <name>SSL/TLS: Renegotiation DoS Vulnerability</name>
    <cvss_base>5.0</cvss_base>
  </nvt>
</result></results></report>`

	findings, _, err := ParseOpenVASXML(strings.NewReader(xml))
	if err != nil {
		t.Fatalf("ParseOpenVASXML: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want 1", len(findings))
	}
	if got := findings[0].OID; got != "1.3.6.1.4.1.25623.1.0.117761" {
		t.Errorf("OpenVAS Finding.OID = %q, want the <nvt oid> attribute", got)
	}

	const js = `{"site":[{"@host":"example.com","@port":"443","@ssl":"true","alerts":[{
	  "pluginid":"40012","alert":"SQL Injection","riskcode":"3","confidence":"2",
	  "desc":"x","solution":"y",
	  "instances":[{"uri":"https://example.com/search","param":"q"}]}]}]}`

	zf, err := ParseZAPJSON(strings.NewReader(js))
	if err != nil {
		t.Fatalf("ParseZAPJSON: %v", err)
	}
	if len(zf) != 1 {
		t.Fatalf("got %d ZAP findings, want 1", len(zf))
	}
	if got := zf[0].OID; got != "40012" {
		t.Errorf("ZAP Finding.OID = %q, want the alert pluginid", got)
	}
}
