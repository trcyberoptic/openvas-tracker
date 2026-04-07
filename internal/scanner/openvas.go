// internal/scanner/openvas.go
package scanner

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Standard report format: <report><results><result>
type ovasReport struct {
	XMLName xml.Name    `xml:"report"`
	Results ovasResults `xml:"results"`
	// GMP nested format: <report><report><results><result>
	Inner *ovasInnerReport `xml:"report"`
}

type ovasInnerReport struct {
	Results   ovasResults `xml:"results"`
	ScanStart string      `xml:"scan_start"`
	ScanEnd   string      `xml:"scan_end"`
}

// GMP envelope: <get_reports_response><report><report><results><result>
type gmpEnvelope struct {
	XMLName xml.Name       `xml:"get_reports_response"`
	Report  gmpOuterReport `xml:"report"`
}

type gmpOuterReport struct {
	Inner gmpInnerReport `xml:"report"`
}

type gmpInnerReport struct {
	Results   ovasResults `xml:"results"`
	ScanStart string      `xml:"scan_start"`
	ScanEnd   string      `xml:"scan_end"`
}

type ovasResults struct {
	Results []ovasResult `xml:"result"`
}

type ovasHost struct {
	IP       string `xml:",chardata"`
	Hostname string `xml:"hostname"`
}

type ovasResult struct {
	Name        string   `xml:"name"`
	Host        ovasHost `xml:"host"`
	Port        string   `xml:"port"`
	Threat      string  `xml:"threat"`
	Severity    float64 `xml:"severity"`
	Description string  `xml:"description"`
	NVT         ovasNVT `xml:"nvt"`
}

type ovasNVT struct {
	OID        string         `xml:"oid,attr"`
	Name       string         `xml:"name"`
	CVSSBase   string         `xml:"cvss_base"`
	CVE        string         `xml:"cve"`
	Solution   string         `xml:"solution"`
	Tags       string         `xml:"tags"`
	Severities ovasSeverities `xml:"severities"`
	Refs       ovasRefs       `xml:"refs"`
}

type ovasRefs struct {
	Ref []ovasRef `xml:"ref"`
}

type ovasRef struct {
	Type string `xml:"type,attr"`
	ID   string `xml:"id,attr"`
}

type ovasSeverities struct {
	Severity []ovasSeverity `xml:"severity"`
}

type ovasSeverity struct {
	Score float64 `xml:"score"`
}

func ParseOpenVASXML(r io.Reader) ([]Finding, *ScanMeta, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read XML: %w", err)
	}

	var rawResults []ovasResult
	var scanStart, scanEnd string

	// Try GMP envelope format first: <get_reports_response>
	if strings.Contains(string(data[:min(500, len(data))]), "get_reports_response") {
		var env gmpEnvelope
		if err := xml.Unmarshal(data, &env); err == nil && len(env.Report.Inner.Results.Results) > 0 {
			rawResults = env.Report.Inner.Results.Results
			scanStart = env.Report.Inner.ScanStart
			scanEnd = env.Report.Inner.ScanEnd
		}
	}

	// Try standard report format: <report>
	if len(rawResults) == 0 {
		var report ovasReport
		if err := xml.Unmarshal(data, &report); err != nil {
			return nil, nil, fmt.Errorf("failed to parse OpenVAS XML: %w", err)
		}
		rawResults = report.Results.Results
		// Check nested <report><report><results>
		if len(rawResults) == 0 && report.Inner != nil {
			rawResults = report.Inner.Results.Results
			if scanStart == "" {
				scanStart = report.Inner.ScanStart
				scanEnd = report.Inner.ScanEnd
			}
		}
	}

	meta := parseScanMeta(scanStart, scanEnd)

	var results []Finding
	for _, res := range rawResults {
		cvss := res.Severity
		if cvss == 0 {
			cvss, _ = strconv.ParseFloat(res.NVT.CVSSBase, 64)
		}
		if cvss == 0 && len(res.NVT.Severities.Severity) > 0 {
			cvss = res.NVT.Severities.Severity[0].Score
		}

		threat := res.Threat
		if threat == "" {
			threat = severityFromCVSS(cvss)
		}

		solution := res.NVT.Solution
		if solution == "" {
			solution = parseTag(res.NVT.Tags, "solution")
		}

		description := res.Description
		if description == "" {
			description = parseTag(res.NVT.Tags, "summary")
		}

		cve := res.NVT.CVE
		if cve == "" || cve == "NOCVE" {
			cve = parseTag(res.NVT.Tags, "cve")
		}
		// Check <refs><ref type="cve"> as fallback (GMP format)
		if (cve == "" || cve == "NOCVE") && len(res.NVT.Refs.Ref) > 0 {
			for _, ref := range res.NVT.Refs.Ref {
				if ref.Type == "cve" && ref.ID != "" {
					cve = ref.ID
					break
				}
			}
		}

		host := strings.TrimSpace(res.Host.IP)
		hostname := strings.TrimSpace(res.Host.Hostname)

		results = append(results, Finding{
			ScanType:    "openvas",
			Title:       res.Name,
			Host:        host,
			Hostname:    hostname,
			Port:        res.Port,
			Severity:    threat,
			CVSSScore:   cvss,
			Description: description,
			Solution:    solution,
			CVEID:       cve,
			OID:         res.NVT.OID,
		})
	}
	return results, meta, nil
}

func parseScanMeta(start, end string) *ScanMeta {
	if start == "" && end == "" {
		return nil
	}
	meta := &ScanMeta{}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02T15:04:05-07:00"} {
		if t, err := time.Parse(layout, strings.TrimSpace(start)); err == nil {
			meta.StartedAt = &t
			break
		}
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02T15:04:05-07:00"} {
		if t, err := time.Parse(layout, strings.TrimSpace(end)); err == nil {
			meta.CompletedAt = &t
			break
		}
	}
	return meta
}

func severityFromCVSS(cvss float64) string {
	switch {
	case cvss >= 9.0:
		return "High" // will be mapped to "critical" by mapSeverity
	case cvss >= 7.0:
		return "High"
	case cvss >= 4.0:
		return "Medium"
	case cvss > 0:
		return "Low"
	default:
		return "Log"
	}
}

func parseTag(tags, key string) string {
	for _, part := range strings.Split(tags, "|") {
		if strings.HasPrefix(part, key+"=") {
			return strings.TrimPrefix(part, key+"=")
		}
	}
	return ""
}
