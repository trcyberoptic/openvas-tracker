// internal/scanner/openvas.go
package scanner

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type OpenVASResult struct {
	Title       string
	Host        string
	Hostname    string
	Port        string
	Severity    string
	CVSSScore   float64
	Description string
	Solution    string
	CVE         string
	OID         string
}

// Standard report format: <report><results><result>
type ovasReport struct {
	XMLName xml.Name    `xml:"report"`
	Results ovasResults `xml:"results"`
	// GMP nested format: <report><report><results><result>
	Inner *ovasInnerReport `xml:"report"`
}

type ovasInnerReport struct {
	Results ovasResults `xml:"results"`
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
	Results ovasResults `xml:"results"`
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

func ParseOpenVASXML(r io.Reader) ([]OpenVASResult, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML: %w", err)
	}

	var rawResults []ovasResult

	// Try GMP envelope format first: <get_reports_response>
	if strings.Contains(string(data[:min(500, len(data))]), "get_reports_response") {
		var env gmpEnvelope
		if err := xml.Unmarshal(data, &env); err == nil && len(env.Report.Inner.Results.Results) > 0 {
			rawResults = env.Report.Inner.Results.Results
		}
	}

	// Try standard report format: <report>
	if len(rawResults) == 0 {
		var report ovasReport
		if err := xml.Unmarshal(data, &report); err != nil {
			return nil, fmt.Errorf("failed to parse OpenVAS XML: %w", err)
		}
		rawResults = report.Results.Results
		// Check nested <report><report><results>
		if len(rawResults) == 0 && report.Inner != nil {
			rawResults = report.Inner.Results.Results
		}
	}

	var results []OpenVASResult
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

		results = append(results, OpenVASResult{
			Title:       res.Name,
			Host:        host,
			Hostname:    hostname,
			Port:        res.Port,
			Severity:    threat,
			CVSSScore:   cvss,
			Description: description,
			Solution:    solution,
			CVE:         cve,
			OID:         res.NVT.OID,
		})
	}
	return results, nil
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
