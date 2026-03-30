package scanner

import (
	"strings"
	"testing"
)

func TestParseZAPJSON(t *testing.T) {
	json := `{
  "@version": "2.16.0",
  "site": [{
    "host": "example.com",
    "port": "443",
    "ssl": "true",
    "alerts": [{
      "pluginid": "40012",
      "alertRef": "40012",
      "alert": "Cross Site Scripting (Reflected)",
      "name": "Cross Site Scripting (Reflected)",
      "riskcode": "3",
      "confidence": "2",
      "riskdesc": "High (Medium)",
      "cweid": "79",
      "wascid": "8",
      "desc": "<p>Cross-site Scripting (XSS) is a vulnerability.</p>",
      "solution": "<p>Validate all input.</p>",
      "instances": [
        {
          "uri": "https://example.com/app/search",
          "method": "GET",
          "param": "q",
          "attack": "<script>alert(1)</script>",
          "evidence": "<script>alert(1)</script>"
        },
        {
          "uri": "https://example.com/app/comment",
          "method": "POST",
          "param": "text",
          "attack": "<img src=x onerror=alert(1)>",
          "evidence": "<img src=x onerror=alert(1)>"
        }
      ]
    }, {
      "pluginid": "10202",
      "alert": "Absence of Anti-CSRF Tokens",
      "riskcode": "2",
      "confidence": "3",
      "cweid": "352",
      "desc": "<p>No Anti-CSRF tokens were found.</p>",
      "solution": "<p>Use anti-CSRF tokens.</p>",
      "instances": [
        {
          "uri": "https://example.com/app/transfer",
          "method": "POST",
          "param": "",
          "evidence": ""
        }
      ]
    }, {
      "pluginid": "10096",
      "alert": "Timestamp Disclosure",
      "riskcode": "0",
      "confidence": "1",
      "cweid": "200",
      "desc": "<p>A timestamp was disclosed.</p>",
      "solution": "<p>Remove timestamps.</p>",
      "instances": [
        {
          "uri": "https://example.com/api/health",
          "method": "GET",
          "param": "",
          "evidence": "1609459200"
        }
      ]
    }]
  }]
}`

	results, err := ParseZAPJSON(strings.NewReader(json))
	if err != nil {
		t.Fatalf("ParseZAPJSON error: %v", err)
	}

	// Info-level findings (riskcode=0) should be skipped
	if len(results) != 3 {
		t.Fatalf("expected 3 results (2 XSS + 1 CSRF, info skipped), got %d", len(results))
	}

	// First XSS instance
	r := results[0]
	if r.Title != "Cross Site Scripting (Reflected)" {
		t.Errorf("title = %q, want %q", r.Title, "Cross Site Scripting (Reflected)")
	}
	if r.Host != "example.com" {
		t.Errorf("host = %q, want %q", r.Host, "example.com")
	}
	if r.Port != "443" {
		t.Errorf("port = %q, want %q", r.Port, "443")
	}
	if r.Severity != "high" {
		t.Errorf("severity = %q, want %q", r.Severity, "high")
	}
	if r.CVSSScore != 7.0 {
		t.Errorf("cvss = %f, want 7.0", r.CVSSScore)
	}
	if r.CWEID != "79" {
		t.Errorf("cweid = %q, want %q", r.CWEID, "79")
	}
	if r.URL != "/app/search" {
		t.Errorf("url = %q, want %q", r.URL, "/app/search")
	}
	if r.Parameter != "q" {
		t.Errorf("parameter = %q, want %q", r.Parameter, "q")
	}
	if r.Evidence != "<script>alert(1)</script>" {
		t.Errorf("evidence = %q, want %q", r.Evidence, "<script>alert(1)</script>")
	}
	if r.Confidence != "medium" {
		t.Errorf("confidence = %q, want %q", r.Confidence, "medium")
	}
	if r.ScanType != "zap" {
		t.Errorf("scanType = %q, want %q", r.ScanType, "zap")
	}
	if r.Description != "Cross-site Scripting (XSS) is a vulnerability." {
		t.Errorf("description HTML not stripped: %q", r.Description)
	}

	// Second XSS instance (different URL)
	r2 := results[1]
	if r2.URL != "/app/comment" {
		t.Errorf("url = %q, want %q", r2.URL, "/app/comment")
	}
	if r2.Parameter != "text" {
		t.Errorf("parameter = %q, want %q", r2.Parameter, "text")
	}

	// CSRF finding
	r3 := results[2]
	if r3.Title != "Absence of Anti-CSRF Tokens" {
		t.Errorf("title = %q, want %q", r3.Title, "Absence of Anti-CSRF Tokens")
	}
	if r3.Severity != "medium" {
		t.Errorf("severity = %q, want %q", r3.Severity, "medium")
	}
	if r3.CVSSScore != 4.0 {
		t.Errorf("cvss = %f, want 4.0", r3.CVSSScore)
	}
	if r3.Confidence != "high" {
		t.Errorf("confidence = %q, want %q", r3.Confidence, "high")
	}
}

func TestParseZAPJSON_Empty(t *testing.T) {
	json := `{"site": []}`
	results, err := ParseZAPJSON(strings.NewReader(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestParseZAPJSON_URLPathExtraction(t *testing.T) {
	json := `{
  "site": [{
    "host": "10.0.0.1",
    "port": "8080",
    "ssl": "false",
    "alerts": [{
      "alert": "SQL Injection",
      "riskcode": "3",
      "confidence": "4",
      "cweid": "89",
      "desc": "SQL injection found",
      "solution": "Use parameterized queries",
      "instances": [{
        "uri": "http://10.0.0.1:8080/api/users?id=1",
        "method": "GET",
        "param": "id",
        "evidence": "error in SQL syntax"
      }]
    }]
  }]
}`

	results, err := ParseZAPJSON(strings.NewReader(json))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].URL != "/api/users" {
		t.Errorf("url = %q, want %q (query string stripped)", results[0].URL, "/api/users")
	}
	if results[0].Host != "10.0.0.1" {
		t.Errorf("host = %q, want %q", results[0].Host, "10.0.0.1")
	}
	if results[0].Confidence != "confirmed" {
		t.Errorf("confidence = %q, want %q", results[0].Confidence, "confirmed")
	}
}
