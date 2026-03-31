package scanner

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
)

type zapReport struct {
	Site []zapSite `json:"site"`
}

type zapSite struct {
	Host   string     `json:"@host"`
	Port   string     `json:"@port"`
	SSL    string     `json:"@ssl"`
	Name   string     `json:"@name"`
	Alerts []zapAlert `json:"alerts"`
}

type zapAlert struct {
	PluginID   string        `json:"pluginid"`
	Alert      string        `json:"alert"`
	RiskCode   string        `json:"riskcode"`
	Confidence string        `json:"confidence"`
	CWEID      string        `json:"cweid"`
	Desc       string        `json:"desc"`
	Solution   string        `json:"solution"`
	Instances  []zapInstance `json:"instances"`
}

type zapInstance struct {
	URI      string `json:"uri"`
	Method   string `json:"method"`
	Param    string `json:"param"`
	Attack   string `json:"attack"`
	Evidence string `json:"evidence"`
}

// ParseZAPJSON parses a ZAP Traditional JSON Report into []Finding.
// Info-level findings (riskcode=0) are skipped.
func ParseZAPJSON(r io.Reader) ([]Finding, error) {
	var report zapReport
	if err := json.NewDecoder(r).Decode(&report); err != nil {
		return nil, fmt.Errorf("failed to parse ZAP JSON: %w", err)
	}

	var results []Finding
	for _, site := range report.Site {
		for _, alert := range site.Alerts {
			severity, cvss := mapZAPRisk(alert.RiskCode)
			if severity == "info" {
				continue
			}

			for _, inst := range alert.Instances {
				urlPath := extractURLPath(inst.URI)

				results = append(results, Finding{
					Host:        site.Host,
					Port:        site.Port,
					Protocol:    "tcp",
					URL:         urlPath,
					Parameter:   inst.Param,
					Title:       alert.Alert,
					Description: stripHTML(alert.Desc),
					Severity:    severity,
					CVSSScore:   cvss,
					CWEID:       alert.CWEID,
					Solution:    stripHTML(alert.Solution),
					Evidence:    inst.Evidence,
					Confidence:  mapZAPConfidence(alert.Confidence),
					ScanType:    "zap",
				})
			}
		}
	}
	return results, nil
}

func mapZAPRisk(code string) (string, float64) {
	switch code {
	case "3":
		return "high", 7.0
	case "2":
		return "medium", 4.0
	case "1":
		return "low", 2.0
	default:
		return "info", 0.0
	}
}

func mapZAPConfidence(code string) string {
	switch code {
	case "4":
		return "confirmed"
	case "3":
		return "high"
	case "2":
		return "medium"
	default:
		return "low"
	}
}

// extractURLPath extracts just the path from a full URI, stripping scheme, host, and query.
func extractURLPath(uri string) string {
	if uri == "" {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	return u.Path
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// stripHTML removes HTML tags and trims whitespace.
func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	return s
}
